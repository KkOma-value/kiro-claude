package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/yourusername/kiro-claude/internal/gateway"
	"github.com/yourusername/kiro-claude/internal/logger"
)

type gatewayClient interface {
	SendRequest(ctx context.Context, req *gateway.CodeWhispererRequest) (*gateway.CodeWhispererResponse, error)
	SendStreamRequest(ctx context.Context, req *gateway.CodeWhispererRequest) (<-chan *gateway.CodeWhispererStreamChunk, <-chan error, error)
	Models() []gateway.ModelInfo
}

// Handler handles HTTP requests for the Anthropic API
type Handler struct {
	gwClient gatewayClient
	logger   logger.Logger
}

// NewHandler creates a new API handler
func NewHandler(gwClient gatewayClient, log logger.Logger) *Handler {
	return &Handler{
		gwClient: gwClient,
		logger:   log,
	}
}

// HandleMessages handles POST /v1/messages requests
func (h *Handler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Errorf("failed to read request body: %v", err)
		h.writeError(w, http.StatusBadRequest, "invalid_request_error", "failed to read request")
		return
	}
	defer r.Body.Close()

	// Parse request
	var req MessageRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Errorf("failed to parse request: %v", err)
		h.writeError(w, http.StatusBadRequest, "invalid_request_error", "invalid request format")
		return
	}

	h.logger.Infof("Handling message request for model=%s, stream=%v", req.Model, req.Stream)

	// Convert to CodeWhisperer format
	cwReq, err := ToCodeWhispererRequest(&req)
	if err != nil {
		h.logger.Errorf("failed to convert request: %v", err)
		h.writeError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	if req.Stream {
		h.handleStreamingRequest(ctx, w, cwReq)
	} else {
		h.handleNonStreamingRequest(ctx, w, cwReq)
	}
}

// handleNonStreamingRequest handles non-streaming message requests
func (h *Handler) handleNonStreamingRequest(ctx context.Context, w http.ResponseWriter, cwReq *gateway.CodeWhispererRequest) {
	// Send request to CodeWhisperer
	cwResp, err := h.gwClient.SendRequest(ctx, cwReq)
	if err != nil {
		h.logger.Errorf("CodeWhisperer request failed: %v", err)
		h.writeGatewayError(w, err)
		return
	}

	// Convert response back to Anthropic format
	resp, err := ToAnthropicResponse(cwResp)
	if err != nil {
		h.logger.Errorf("failed to convert response: %v", err)
		h.writeError(w, http.StatusInternalServerError, "api_error", "response conversion failed")
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Errorf("failed to write response: %v", err)
	}
}

// handleStreamingRequest handles streaming message requests
func (h *Handler) handleStreamingRequest(ctx context.Context, w http.ResponseWriter, cwReq *gateway.CodeWhispererRequest) {
	// Get stream from gateway
	chunks, errs, err := h.gwClient.SendStreamRequest(ctx, cwReq)
	if err != nil {
		h.logger.Errorf("failed to start stream: %v", err)
		h.writeGatewayError(w, err)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.Errorf("response writer does not support flushing")
		h.writeError(w, http.StatusInternalServerError, "api_error", "streaming not supported")
		return
	}

	chunkCh := chunks
	errCh := errs

	// Stream chunks to client
	for chunkCh != nil || errCh != nil {
		select {
		case chunk, ok := <-chunkCh:
			if !ok {
				chunkCh = nil
				continue
			}

			// Convert chunk to Anthropic format
			event, err := ToAnthropicStreamEvent(chunk)
			if err != nil {
				h.logger.Warnf("failed to convert stream chunk: %v", err)
				continue
			}

			// Marshal event
			eventJSON, err := json.Marshal(event)
			if err != nil {
				h.logger.Warnf("failed to marshal event: %v", err)
				continue
			}

			// Write SSE format: "data: {json}\n\n"
			if event.Type != "" {
				_, _ = w.Write([]byte("event: "))
				_, _ = w.Write([]byte(event.Type))
				_, _ = w.Write([]byte("\n"))
			}
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(eventJSON)
			_, _ = w.Write([]byte("\n\n"))

			flusher.Flush()

		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			if err != nil {
				h.logger.Errorf("stream error: %v", err)
			}
			return

		case <-ctx.Done():
			h.logger.Infof("stream context cancelled")
			return
		}
	}

	h.logger.Infof("stream completed")
}

// HandleModels handles GET /v1/models request
func (h *Handler) HandleModels(w http.ResponseWriter, r *http.Request) {
	backendModels := h.gwClient.Models()
	models := make([]ModelData, 0, len(backendModels))
	for _, model := range backendModels {
		models = append(models, ModelData{
			ID:      model.ID,
			Object:  "model",
			OwnedBy: model.Provider,
		})
	}

	resp := ModelsResponse{
		Data:   models,
		Object: "list",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Errorf("failed to write response: %v", err)
	}
}

// HandleHealth handles GET /health request
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) writeGatewayError(w http.ResponseWriter, err error) {
	var apiErr *gateway.APIError
	if errors.As(err, &apiErr) {
		errorType := "api_error"
		if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			errorType = "invalid_request_error"
		}
		h.writeError(w, apiErr.StatusCode, errorType, apiErr.Error())
		return
	}

	h.writeError(w, http.StatusBadGateway, "api_error", "gateway request failed")
}

func (h *Handler) writeError(w http.ResponseWriter, status int, errorType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"type": "error",
		"error": ErrorResponse{
			Type:    errorType,
			Message: message,
		},
	})
}
