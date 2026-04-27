package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/yourusername/kiro-claude/internal/auth"
	"github.com/yourusername/kiro-claude/internal/config"
	"github.com/yourusername/kiro-claude/internal/logger"
)

// APIError wraps upstream HTTP failures with status information.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("CodeWhisperer error: HTTP %d", e.StatusCode)
	}
	return e.Message
}

// Client wraps HTTP communication with CodeWhisperer API
type Client struct {
	endpoint string
	tokenMgr *auth.TokenManager
	client   *http.Client
	logger   logger.Logger
}

// NewClient creates a new CodeWhisperer client
func NewClient(tokenMgr *auth.TokenManager, endpoint string, proxyCfg config.ProxyConfig, log logger.Logger) *Client {
	transport := &http.Transport{}
	if proxyURL := firstProxyURL(proxyCfg); proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &Client{
		endpoint: endpoint,
		tokenMgr: tokenMgr,
		logger:   log,
		client: &http.Client{
			Timeout:   120 * time.Second, // Longer timeout for streaming
			Transport: transport,
		},
	}
}

// Models returns the supported Anthropic-facing models for this backend.
func (c *Client) Models() []ModelInfo {
	return SupportedModels()
}

// SendRequest sends a request to CodeWhisperer API and returns the response
func (c *Client) SendRequest(ctx context.Context, req *CodeWhispererRequest) (*CodeWhispererResponse, error) {
	token, err := c.tokenMgr.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/api/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("User-Agent", "kiro-claude/1.0")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		c.logger.Warnf("CodeWhisperer request failed with status %d: %s", resp.StatusCode, string(respBody))

		// Try to parse error response
		var errResp struct {
			Error *ErrorBlock `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error != nil {
			return nil, &APIError{
				StatusCode: resp.StatusCode,
				Message:    errResp.Error.Message,
			}
		}
		return nil, &APIError{StatusCode: resp.StatusCode}
	}

	// Parse successful response
	var cwResp CodeWhispererResponse
	if err := json.Unmarshal(respBody, &cwResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &cwResp, nil
}

// SendStreamRequest sends a request with streaming enabled and returns a channel of chunks
func (c *Client) SendStreamRequest(ctx context.Context, req *CodeWhispererRequest) (<-chan *CodeWhispererStreamChunk, <-chan error, error) {
	token, err := c.tokenMgr.GetAccessToken(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get access token: %w", err)
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/api/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("User-Agent", "kiro-claude/1.0")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	// Check status code before streaming
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.Warnf("CodeWhisperer stream request failed with status %d: %s", resp.StatusCode, string(respBody))
		var errResp struct {
			Error *ErrorBlock `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error != nil {
			return nil, nil, &APIError{
				StatusCode: resp.StatusCode,
				Message:    errResp.Error.Message,
			}
		}
		return nil, nil, &APIError{StatusCode: resp.StatusCode}
	}

	// Create channels for streaming
	chunkChan := make(chan *CodeWhispererStreamChunk, 10)
	errChan := make(chan error, 1)

	// Start goroutine to read stream
	go c.readStream(resp.Body, chunkChan, errChan)

	return chunkChan, errChan, nil
}

func firstProxyURL(proxyCfg config.ProxyConfig) *url.URL {
	candidates := []string{
		proxyCfg.SOCKS5Proxy,
		proxyCfg.HTTPSProxy,
		proxyCfg.HTTPProxy,
	}

	for _, rawURL := range candidates {
		if rawURL == "" {
			continue
		}
		proxyURL, err := url.Parse(rawURL)
		if err == nil {
			return proxyURL
		}
	}

	return nil
}

// readStream reads SSE stream and sends chunks to the channel
func (c *Client) readStream(body io.ReadCloser, chunks chan<- *CodeWhispererStreamChunk, errs chan<- error) {
	defer body.Close()
	defer close(chunks)
	defer close(errs)

	reader := NewEventReader(body)
	for {
		line, err := reader.ReadLine()
		if err != nil {
			if err != io.EOF {
				errs <- fmt.Errorf("stream read error: %w", err)
			}
			return
		}

		if len(line) == 0 {
			continue // Empty line, skip
		}

		// Parse SSE "data: {json}" format
		if bytes.HasPrefix(line, []byte("data: ")) {
			jsonData := bytes.TrimPrefix(line, []byte("data: "))

			var chunk CodeWhispererStreamChunk
			if err := json.Unmarshal(jsonData, &chunk); err != nil {
				errs <- fmt.Errorf("failed to parse chunk: %w", err)
				continue
			}

			chunks <- &chunk
		}
	}
}

// NewEventReader creates a reader for SSE event streams
// This is a helper for reading line-by-line from SSE
func NewEventReader(r io.Reader) *EventReader {
	return &EventReader{
		reader: r,
		buf:    make([]byte, 4096),
	}
}

// EventReader reads lines from an SSE stream
type EventReader struct {
	reader io.Reader
	buf    []byte
	pos    int
	end    int
}

// ReadLine reads the next line from the stream
func (er *EventReader) ReadLine() ([]byte, error) {
	for {
		// Look for newline in buffer
		for i := er.pos; i < er.end; i++ {
			if er.buf[i] == '\n' {
				line := er.buf[er.pos:i]
				er.pos = i + 1
				return line, nil
			}
		}

		// No newline found, read more data
		if er.pos > 0 {
			// Move remaining data to start of buffer
			copy(er.buf, er.buf[er.pos:er.end])
			er.end -= er.pos
			er.pos = 0
		}

		n, err := er.reader.Read(er.buf[er.end:])
		if err != nil {
			if er.end > er.pos {
				// Return remaining data
				line := er.buf[er.pos:er.end]
				er.end = 0
				er.pos = 0
				return line, nil
			}
			return nil, err
		}

		er.end += n
	}
}
