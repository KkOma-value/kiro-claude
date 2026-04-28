package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yourusername/kiro-claude/internal/gateway"
	"github.com/yourusername/kiro-claude/internal/logger"
)

type stubBackendClient struct {
	sendRequest       func(ctx context.Context, req *gateway.KiroRequest) (string, []gateway.KiroStreamEvent, error)
	sendStreamRequest func(ctx context.Context, req *gateway.KiroRequest) (<-chan gateway.KiroStreamEvent, <-chan error, error)
	models            func() []gateway.ModelInfo
}

func (s *stubBackendClient) SendRequest(ctx context.Context, req *gateway.KiroRequest) (string, []gateway.KiroStreamEvent, error) {
	return s.sendRequest(ctx, req)
}

func (s *stubBackendClient) SendStreamRequest(ctx context.Context, req *gateway.KiroRequest) (<-chan gateway.KiroStreamEvent, <-chan error, error) {
	return s.sendStreamRequest(ctx, req)
}

func (s *stubBackendClient) Models() []gateway.ModelInfo {
	if s.models != nil {
		return s.models()
	}
	return gateway.SupportedModels()
}

type flushRecorder struct {
	*httptest.ResponseRecorder
}

func (r *flushRecorder) Flush() {}

func TestHandleMessagesReturnsAnthropicErrorEnvelope(t *testing.T) {
	handler := NewHandler(&stubBackendClient{
		sendRequest: func(ctx context.Context, req *gateway.KiroRequest) (string, []gateway.KiroStreamEvent, error) {
			return "", nil, &gateway.APIError{StatusCode: http.StatusUnauthorized, Message: "bad token"}
		},
		sendStreamRequest: func(ctx context.Context, req *gateway.KiroRequest) (<-chan gateway.KiroStreamEvent, <-chan error, error) {
			return nil, nil, nil
		},
	}, logger.NewSimpleLogger(logger.LevelError))

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{
		"model":"claude-sonnet-4-5",
		"messages":[{"role":"user","content":"hello"}],
		"max_tokens":32
	}`))
	rec := httptest.NewRecorder()

	handler.HandleMessages(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"type":"error"`) || !strings.Contains(body, `"invalid_request_error"`) {
		t.Fatalf("unexpected error body: %s", body)
	}
}

func TestHandleMessagesStreamingWritesAnthropicSSE(t *testing.T) {
	handler := NewHandler(&stubBackendClient{
		sendRequest: func(ctx context.Context, req *gateway.KiroRequest) (string, []gateway.KiroStreamEvent, error) {
			return "", nil, nil
		},
		sendStreamRequest: func(ctx context.Context, req *gateway.KiroRequest) (<-chan gateway.KiroStreamEvent, <-chan error, error) {
			eventCh := make(chan gateway.KiroStreamEvent, 2)
			errCh := make(chan error, 1)
			eventCh <- gateway.KiroStreamEvent{Content: "hello world"}
			close(eventCh)
			close(errCh)
			return eventCh, errCh, nil
		},
	}, logger.NewSimpleLogger(logger.LevelError))

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{
		"model":"claude-sonnet-4-5",
		"messages":[{"role":"user","content":"hello"}],
		"max_tokens":32,
		"stream":true
	}`))
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}

	handler.HandleMessages(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "event: message_start") {
		t.Fatalf("missing message_start SSE event: %s", body)
	}
	if !strings.Contains(body, "event: content_block_start") {
		t.Fatalf("missing content_block_start SSE event: %s", body)
	}
	if !strings.Contains(body, `"text_delta"`) {
		t.Fatalf("missing text_delta in stream: %s", body)
	}
	if !strings.Contains(body, "event: message_stop") {
		t.Fatalf("missing message_stop SSE event: %s", body)
	}
}

func TestHandleMessagesNonStreamingReturnsJSON(t *testing.T) {
	handler := NewHandler(&stubBackendClient{
		sendRequest: func(ctx context.Context, req *gateway.KiroRequest) (string, []gateway.KiroStreamEvent, error) {
			return "Hello from Kiro!", nil, nil
		},
		sendStreamRequest: func(ctx context.Context, req *gateway.KiroRequest) (<-chan gateway.KiroStreamEvent, <-chan error, error) {
			return nil, nil, nil
		},
	}, logger.NewSimpleLogger(logger.LevelError))

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{
		"model":"claude-sonnet-4-5",
		"messages":[{"role":"user","content":"hello"}],
		"max_tokens":32
	}`))
	rec := httptest.NewRecorder()

	handler.HandleMessages(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"role":"assistant"`) {
		t.Fatalf("missing assistant role: %s", body)
	}
	if !strings.Contains(body, "Hello from Kiro!") {
		t.Fatalf("missing response text: %s", body)
	}
}
