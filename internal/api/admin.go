package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/yourusername/kiro-claude/internal/auth"
	"github.com/yourusername/kiro-claude/internal/config"
	"github.com/yourusername/kiro-claude/internal/logger"
	"github.com/yourusername/kiro-claude/internal/model"
)

const Version = "2.0.0"

// AdminHandler serves management API endpoints for the web frontend.
type AdminHandler struct {
	cfg         *config.Config
	resolver    *model.Resolver
	tokenMgr    *auth.TokenManager
	log         logger.Logger
	startTime   time.Time
	bindAddress string
}

// NewAdminHandler creates a new admin API handler.
// tokenMgr may be nil when running without credentials.
func NewAdminHandler(
	cfg *config.Config,
	resolver *model.Resolver,
	tokenMgr *auth.TokenManager,
	log logger.Logger,
	bindAddress string,
) *AdminHandler {
	return &AdminHandler{
		cfg:         cfg,
		resolver:    resolver,
		tokenMgr:    tokenMgr,
		log:         log,
		startTime:   time.Now(),
		bindAddress: bindAddress,
	}
}

// --- GET /api/status ---

type StatusResponse struct {
	Mode               string        `json:"mode"`
	BindAddress        string        `json:"bind_address"`
	AuthGuard          bool          `json:"auth_guard"`
	AuthGuardMaskedKey string        `json:"auth_guard_masked_key,omitempty"`
	DebugDump          bool          `json:"debug_dump"`
	Accounts           []AccountInfo `json:"accounts"`
	UpstreamEndpoint   string        `json:"upstream_endpoint"`
	UptimeSeconds      int64         `json:"uptime_seconds"`
	Version            string        `json:"version"`
}

type AccountInfo struct {
	ID        string `json:"id"`
	AuthType  string `json:"auth_type"`
	Healthy   bool   `json:"healthy"`
	Failures  int    `json:"failures"`
	IsCurrent bool   `json:"is_current"`
}

func (h *AdminHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	var accounts []AccountInfo
	if h.tokenMgr != nil {
		status := h.tokenMgr.Status()
		for i := 0; i < status.TotalAccounts; i++ {
			authType := status.AuthMethod
			if authType == "" {
				authType = "kiro-desktop"
			}
			accounts = append(accounts, AccountInfo{
				ID:        authType + "-" + strings.Repeat("x", 1),
				AuthType:  authType,
				Healthy:   !status.IsExpired,
				Failures:  0,
				IsCurrent: i == status.CurrentIndex,
			})
		}
	}
	if accounts == nil {
		accounts = []AccountInfo{}
	}

	maskedKey := ""
	if h.cfg.Security.ProxyAPIKey != "" {
		maskedKey = maskKey(h.cfg.Security.ProxyAPIKey)
	}

	resp := StatusResponse{
		Mode:               h.cfg.Runtime.Mode,
		BindAddress:        h.bindAddress,
		AuthGuard:          h.cfg.Security.ProxyAPIKey != "",
		AuthGuardMaskedKey: maskedKey,
		DebugDump:          h.cfg.Logging.DebugDump,
		Accounts:           accounts,
		UpstreamEndpoint:   h.cfg.Runtime.UpstreamEndpoint,
		UptimeSeconds:      int64(time.Since(h.startTime).Seconds()),
		Version:            Version,
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- POST /api/resolve-model ---

type ResolveRequest struct {
	Model string `json:"model"`
}

type ResolveResponse struct {
	Original   string         `json:"original"`
	Normalized string         `json:"normalized"`
	InternalID string         `json:"internal_id"`
	Source     string         `json:"source"`
	Pipeline   []PipelineStep `json:"pipeline"`
}

type PipelineStep struct {
	Step   string `json:"step"`
	Input  string `json:"input"`
	Output string `json:"output"`
}

func (h *AdminHandler) HandleResolveModel(w http.ResponseWriter, r *http.Request) {
	var req ResolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Model == "" {
		writeAdminError(w, http.StatusBadRequest, "model field is required")
		return
	}

	normalized := model.NormalizeModelName(req.Model)
	resolution := h.resolver.Resolve(req.Model)

	pipeline := []PipelineStep{
		{
			Step:   "normalize",
			Input:  req.Model,
			Output: normalized,
		},
	}

	if resolution.Source == "static" {
		pipeline = append(pipeline, PipelineStep{
			Step:   "static_lookup",
			Input:  normalized,
			Output: resolution.InternalID,
		})
	} else {
		pipeline = append(pipeline, PipelineStep{
			Step:   "passthrough",
			Input:  normalized,
			Output: normalized,
		})
	}

	resp := ResolveResponse{
		Original:   resolution.Original,
		Normalized: resolution.ExternalName,
		InternalID: resolution.InternalID,
		Source:     resolution.Source,
		Pipeline:   pipeline,
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- GET /api/endpoints ---

type EndpointsResponse struct {
	Endpoints []EndpointInfo `json:"endpoints"`
}

type EndpointInfo struct {
	Path      string `json:"path"`
	Method    string `json:"method"`
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
}

func (h *AdminHandler) HandleEndpoints(w http.ResponseWriter, r *http.Request) {
	endpoints := []EndpointInfo{
		h.checkEndpoint("GET", "/health"),
		h.checkEndpoint("GET", "/v1/models"),
	}

	writeJSON(w, http.StatusOK, EndpointsResponse{Endpoints: endpoints})
}

func (h *AdminHandler) checkEndpoint(method, path string) EndpointInfo {
	start := time.Now()
	status := "ok"

	client := &http.Client{Timeout: 5 * time.Second}
	reqURL := "http://" + h.bindAddress + path

	resp, err := client.Get(reqURL)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		status = "error"
	} else {
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			status = "error"
		} else if resp.StatusCode >= 400 {
			status = "degraded"
		}
	}

	return EndpointInfo{
		Path:      path,
		Method:    method,
		Status:    status,
		LatencyMs: latency,
	}
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeAdminError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func maskKey(key string) string {
	if len(key) <= 6 {
		return "***"
	}
	return key[:3] + strings.Repeat("*", len(key)-6) + key[len(key)-3:]
}
