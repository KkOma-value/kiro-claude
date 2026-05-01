package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/kiro-claude/internal/gateway"
	"github.com/yourusername/kiro-claude/internal/logger"
)

type backendClient interface {
	SendRequest(ctx context.Context, req *gateway.KiroRequest) (string, []gateway.KiroStreamEvent, error)
	SendStreamRequest(ctx context.Context, req *gateway.KiroRequest) (<-chan gateway.KiroStreamEvent, <-chan error, error)
	Models() []gateway.ModelInfo
}

// Handler handles HTTP requests for the Anthropic API
type Handler struct {
	client backendClient
	logger logger.Logger
}

// NewHandler creates a new API handler
func NewHandler(client backendClient, log logger.Logger) *Handler {
	return &Handler{
		client: client,
		logger: log,
	}
}

// HandleMessages handles POST /v1/messages requests
func (h *Handler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Errorf("failed to read request body: %v", err)
		h.writeError(w, http.StatusBadRequest, "invalid_request_error", "failed to read request")
		return
	}
	defer r.Body.Close()

	var req MessageRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Errorf("failed to parse request: %v", err)
		h.writeError(w, http.StatusBadRequest, "invalid_request_error", "invalid request format")
		return
	}

	h.logger.Infof("Handling message request model=%s stream=%v messages=%d tools=%d", req.Model, req.Stream, len(req.Messages), len(req.Tools))

	kiroReq, err := ToKiroRequest(&req)
	if err != nil {
		h.logger.Errorf("failed to convert request: %v", err)
		h.writeError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
	defer cancel()

	if req.Stream {
		h.handleStreaming(ctx, w, kiroReq, req.Model)
	} else {
		h.handleNonStreaming(ctx, w, kiroReq, req.Model)
	}
}

// handleNonStreaming sends a non-streaming request and returns a complete Anthropic MessageResponse.
func (h *Handler) handleNonStreaming(ctx context.Context, w http.ResponseWriter, kiroReq *gateway.KiroRequest, requestModel string) {
	text, toolEvents, err := h.client.SendRequest(ctx, kiroReq)
	if err != nil {
		h.logger.Errorf("Kiro request failed: %v", err)
		h.writeGatewayError(w, err)
		return
	}

	// Build Anthropic response from Kiro results
	var content []ContentBlock
	if text != "" {
		content = append(content, ContentBlock{Type: "text", Text: text})
	}

	stopReason := "end_turn"
	for _, evt := range toolEvents {
		if evt.Name != "" && evt.ToolUseID != "" {
			var input json.RawMessage
			if evt.Input != "" {
				input = json.RawMessage(evt.Input)
			} else {
				input = json.RawMessage(`{}`)
			}
			content = append(content, ContentBlock{
				Type:  "tool_use",
				ID:    evt.ToolUseID,
				Name:  evt.Name,
				Input: input,
			})
			stopReason = "tool_use"
		}
	}

	if len(content) == 0 {
		content = append(content, ContentBlock{Type: "text", Text: ""})
	}

	resp := MessageResponse{
		ID:         "msg_" + uuid.New().String()[:8],
		Type:       "message",
		Role:       "assistant",
		Model:      requestModel,
		Content:    content,
		StopReason: stopReason,
		Usage:      UsageBlock{InputTokens: 0, OutputTokens: 0},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Errorf("failed to write response: %v", err)
	}
}

// handleStreaming converts Kiro SSE events to Anthropic SSE format in real time.
func (h *Handler) handleStreaming(ctx context.Context, w http.ResponseWriter, kiroReq *gateway.KiroRequest, requestModel string) {
	eventCh, errCh, err := h.client.SendStreamRequest(ctx, kiroReq)
	if err != nil {
		h.logger.Errorf("failed to start stream: %v", err)
		h.writeGatewayError(w, err)
		return
	}

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

	state := &StreamState{Model: requestModel}

	// Send message_start
	h.writeSSE(w, flusher, state.BuildMessageStart(requestModel))

	evCh := eventCh
	erCh := errCh

	for evCh != nil || erCh != nil {
		select {
		case evt, ok := <-evCh:
			if !ok {
				evCh = nil
				continue
			}
			for _, sseEvent := range state.ProcessKiroEvent(evt) {
				h.writeSSE(w, flusher, sseEvent)
			}

		case err, ok := <-erCh:
			if !ok {
				erCh = nil
				continue
			}
			if err != nil {
				h.logger.Errorf("stream error: %v", err)
			}
			// Finalize even on error so Claude Code gets a proper stream end
			for _, sseEvent := range state.Finalize() {
				h.writeSSE(w, flusher, sseEvent)
			}
			return

		case <-ctx.Done():
			h.logger.Infof("stream context cancelled")
			for _, sseEvent := range state.Finalize() {
				h.writeSSE(w, flusher, sseEvent)
			}
			return
		}
	}

	// Normal completion — finalize the stream
	for _, sseEvent := range state.Finalize() {
		h.writeSSE(w, flusher, sseEvent)
	}

	h.logger.Infof("stream completed")
}

func (h *Handler) writeSSE(w http.ResponseWriter, flusher http.Flusher, event StreamEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		h.logger.Warnf("failed to marshal SSE event: %v", err)
		return
	}

	if event.Type != "" {
		_, _ = w.Write([]byte("event: " + event.Type + "\n"))
	}
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(data)
	_, _ = w.Write([]byte("\n\n"))
	flusher.Flush()
}

// HandleModels handles GET /v1/models request
func (h *Handler) HandleModels(w http.ResponseWriter, r *http.Request) {
	backendModels := h.client.Models()
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
