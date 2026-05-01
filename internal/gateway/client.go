package gateway

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/kiro-claude/internal/auth"
	"github.com/yourusername/kiro-claude/internal/config"
	"github.com/yourusername/kiro-claude/internal/logger"
)

const (
	kiroVersion = "0.11.63"
)

// Client wraps HTTP communication with Kiro generateAssistantResponse API
type Client struct {
	tokenMgr *auth.TokenManager
	client   *http.Client
	logger   logger.Logger
}

// NewClient creates a new Kiro gateway client
func NewClient(tokenMgr *auth.TokenManager, proxyCfg config.ProxyConfig, log logger.Logger) *Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}
	if proxyURL := firstProxyURL(proxyCfg); proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &Client{
		tokenMgr: tokenMgr,
		logger:   log,
		client: &http.Client{
			Timeout:   180 * time.Second,
			Transport: transport,
		},
	}
}

func (c *Client) Models() []ModelInfo {
	return SupportedModels()
}

// SendRequest sends a non-streaming request to Kiro and returns the full response text + tool calls
func (c *Client) SendRequest(ctx context.Context, req *KiroRequest) (string, []KiroStreamEvent, error) {
	body, err := c.doRequest(ctx, req)
	if err != nil {
		return "", nil, err
	}
	defer body.Close()

	raw, err := io.ReadAll(body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read response: %w", err)
	}

	return c.parseFullResponse(raw)
}

// SendStreamRequest sends a streaming request and returns a channel of parsed events
func (c *Client) SendStreamRequest(ctx context.Context, req *KiroRequest) (<-chan KiroStreamEvent, <-chan error, error) {
	body, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	eventCh := make(chan KiroStreamEvent, 32)
	errCh := make(chan error, 1)

	go c.readKiroStream(body, eventCh, errCh)

	return eventCh, errCh, nil
}

func (c *Client) Close() {}

// doRequest builds and sends the HTTP request to Kiro
func (c *Client) doRequest(ctx context.Context, req *KiroRequest) (io.ReadCloser, error) {
	token, err := c.tokenMgr.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	cred := c.tokenMgr.CurrentCredential()
	region := cred.EffectiveRegion()
	endpoint := fmt.Sprintf("https://q.%s.amazonaws.com/generateAssistantResponse", region)

	// Set profileArn for social auth
	if !cred.IsIDC() && cred.ProfileArn != "" {
		req.ProfileArn = cred.ProfileArn
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set required headers matching KiroIDE format
	machineID := generateMachineID(cred)
	osName := getOSName()
	goVersion := strings.TrimPrefix(runtime.Version(), "go")

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("amz-sdk-invocation-id", uuid.New().String())
	httpReq.Header.Set("amz-sdk-request", "attempt=1; max=3")
	httpReq.Header.Set("x-amzn-codewhisperer-optout", "true")
	httpReq.Header.Set("x-amzn-kiro-agent-mode", "vibe")
	httpReq.Header.Set("x-amz-user-agent", fmt.Sprintf("aws-sdk-js/1.0.34 KiroIDE-%s-%s", kiroVersion, machineID))
	httpReq.Header.Set("User-Agent", fmt.Sprintf("aws-sdk-js/1.0.34 ua/2.1 os/%s lang/go md/go#%s api/codewhispererstreaming#1.0.34 m/E KiroIDE-%s-%s", osName, goVersion, kiroVersion, machineID))

	c.logger.Debugf("Sending request to %s (region=%s, auth=%s)", endpoint, region, cred.AuthMethod)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.Warnf("Kiro request failed with status %d: %s", resp.StatusCode, string(respBody))
		return nil, &APIError{StatusCode: resp.StatusCode, Message: string(respBody)}
	}

	return resp.Body, nil
}

// readKiroStream parses the Kiro custom SSE format: :message-typeevent{json}
func (c *Client) readKiroStream(body io.ReadCloser, events chan<- KiroStreamEvent, errs chan<- error) {
	defer body.Close()
	defer close(events)
	defer close(errs)

	buf := make([]byte, 0, 65536)
	readBuf := make([]byte, 8192)

	for {
		n, err := body.Read(readBuf)
		if n > 0 {
			buf = append(buf, readBuf[:n]...)

			// Parse all complete events from buffer
			buf = c.extractAndSendEvents(buf, events)
		}
		if err != nil {
			if err != io.EOF {
				errs <- fmt.Errorf("stream read error: %w", err)
			}
			// Process any remaining data
			if len(buf) > 0 {
				c.extractAndSendEvents(buf, events)
			}
			return
		}
	}
}

// sseEventRegex matches :message-typeevent followed by a JSON object
var sseEventRegex = regexp.MustCompile(`:message-typeevent(\{[^}]*\})`)

// extractAndSendEvents parses Kiro SSE events from buffer and sends them to channel.
// Returns remaining unparsed bytes.
func (c *Client) extractAndSendEvents(buf []byte, events chan<- KiroStreamEvent) []byte {
	data := string(buf)

	matches := sseEventRegex.FindAllStringSubmatchIndex(data, -1)
	if len(matches) == 0 {
		return buf
	}

	lastEnd := 0
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		jsonStart := match[2]
		jsonEnd := match[3]
		jsonStr := data[jsonStart:jsonEnd]

		var evt KiroStreamEvent
		if err := json.Unmarshal([]byte(jsonStr), &evt); err != nil {
			c.logger.Debugf("Failed to parse SSE event JSON: %v", err)
			lastEnd = match[1]
			continue
		}

		// Skip followup prompts
		if evt.FollowupPrompt != "" {
			lastEnd = match[1]
			continue
		}

		events <- evt
		lastEnd = match[1]
	}

	if lastEnd > 0 && lastEnd <= len(buf) {
		return buf[lastEnd:]
	}
	return buf
}

// parseFullResponse parses a non-streaming Kiro response (same SSE format but all at once)
func (c *Client) parseFullResponse(raw []byte) (string, []KiroStreamEvent, error) {
	data := string(raw)
	var fullContent strings.Builder
	var toolEvents []KiroStreamEvent

	matches := sseEventRegex.FindAllStringSubmatch(data, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		var evt KiroStreamEvent
		if err := json.Unmarshal([]byte(match[1]), &evt); err != nil {
			continue
		}
		if evt.FollowupPrompt != "" {
			continue
		}
		if evt.Name != "" && evt.ToolUseID != "" {
			toolEvents = append(toolEvents, evt)
		} else if evt.Content != "" {
			// Unescape \n in content
			content := strings.ReplaceAll(evt.Content, `\n`, "\n")
			fullContent.WriteString(content)
		}
	}

	return fullContent.String(), toolEvents, nil
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

func generateMachineID(cred auth.Credential) string {
	key := cred.ProfileArn
	if key == "" {
		key = cred.ClientID
	}
	if key == "" {
		key = "KIRO_DEFAULT_MACHINE"
	}
	h := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", h)
}

func getOSName() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	case "windows":
		return "windows"
	default:
		return runtime.GOOS
	}
}

// hostname returns the machine hostname or a fallback
func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
