package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// httpStatusText returns the standard status text for an HTTP status code.
func httpStatusText(code int) string {
	text := http.StatusText(code)
	if text == "" {
		return "Unknown"
	}
	return text
}

// mimeTypeFromHeaders extracts the MIME type from a Content-Type header, defaulting to text/plain.
func mimeTypeFromHeaders(headers map[string]string) string {
	for k, v := range headers {
		if strings.EqualFold(k, "content-type") {
			// Strip charset etc: "text/html; charset=utf-8" → "text/html"
			if idx := strings.IndexByte(v, ';'); idx >= 0 {
				return strings.TrimSpace(v[:idx])
			}
			return v
		}
	}
	return "text/plain"
}

// isAlreadyEnabledErr checks if the error is "Interception already enabled/disabled",
// which happens when a previous session left state on WebKit's side.
func isAlreadyEnabledErr(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "already enabled") ||
		strings.Contains(err.Error(), "already disabled"))
}

// NetworkMonitor collects network requests and responses.
type NetworkMonitor struct {
	mu       sync.Mutex
	requests map[string]*CapturedRequest
	started  bool
}

// CapturedRequest pairs a network request with its optional response and completion status.
type CapturedRequest struct {
	Request  webkit.NetworkRequest   `json:"request"`
	Response *webkit.NetworkResponse `json:"response,omitempty"`
	Done     bool                    `json:"done"`
}

// NewNetworkMonitor creates a new NetworkMonitor.
func NewNetworkMonitor() *NetworkMonitor {
	return &NetworkMonitor{
		requests: make(map[string]*CapturedRequest),
	}
}

// Start enables network monitoring and registers event handlers on the client.
// It is idempotent: calling Start multiple times will not register duplicate handlers.
func (m *NetworkMonitor) Start(ctx context.Context, client *webkit.Client) error {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return nil
	}
	m.started = true
	m.mu.Unlock()

	_, err := client.Send(ctx, "Network.enable", nil)
	if err != nil {
		m.mu.Lock()
		m.started = false
		m.mu.Unlock()
		return err
	}

	client.OnEvent("Network.requestWillBeSent", func(method string, params json.RawMessage) {
		var evt struct {
			RequestID string                `json:"requestId"`
			Request   webkit.NetworkRequest `json:"request"`
		}
		if err := json.Unmarshal(params, &evt); err != nil {
			return
		}
		// requestId is at the top level of the event, not inside request.
		evt.Request.RequestID = evt.RequestID
		m.mu.Lock()
		if len(m.requests) >= maxCollectorEntries {
			m.mu.Unlock()
			return
		}
		m.requests[evt.RequestID] = &CapturedRequest{
			Request: evt.Request,
		}
		m.mu.Unlock()
	})

	client.OnEvent("Network.responseReceived", func(method string, params json.RawMessage) {
		var evt struct {
			RequestID string                 `json:"requestId"`
			Response  webkit.NetworkResponse `json:"response"`
		}
		if err := json.Unmarshal(params, &evt); err != nil {
			return
		}
		m.mu.Lock()
		if req, ok := m.requests[evt.RequestID]; ok {
			req.Response = &evt.Response
		}
		m.mu.Unlock()
	})

	client.OnEvent("Network.loadingFinished", func(method string, params json.RawMessage) {
		var evt struct {
			RequestID string `json:"requestId"`
		}
		if err := json.Unmarshal(params, &evt); err != nil {
			return
		}
		m.mu.Lock()
		if req, ok := m.requests[evt.RequestID]; ok {
			req.Done = true
		}
		m.mu.Unlock()
	})

	client.OnEvent("Network.loadingFailed", func(method string, params json.RawMessage) {
		var evt struct {
			RequestID string `json:"requestId"`
		}
		if err := json.Unmarshal(params, &evt); err != nil {
			return
		}
		m.mu.Lock()
		if req, ok := m.requests[evt.RequestID]; ok {
			req.Done = true
		}
		m.mu.Unlock()
	})

	return nil
}

// Stop disables network monitoring.
func (m *NetworkMonitor) Stop(ctx context.Context, client *webkit.Client) error {
	m.mu.Lock()
	m.started = false
	m.mu.Unlock()
	_, err := client.Send(ctx, "Network.disable", nil)
	return err
}

// GetRequests returns a copy of all collected requests.
func (m *NetworkMonitor) GetRequests() []CapturedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]CapturedRequest, 0, len(m.requests))
	for _, req := range m.requests {
		result = append(result, *req)
	}
	return result
}

// GetResponseBody retrieves the body of a network response by request ID.
func GetResponseBody(ctx context.Context, client *webkit.Client, requestID string) (string, bool, error) {
	result, err := client.Send(ctx, "Network.getResponseBody", map[string]string{
		"requestId": requestID,
	})
	if err != nil {
		return "", false, err
	}

	var resp struct {
		Body          string `json:"body"`
		Base64Encoded bool   `json:"base64Encoded"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", false, fmt.Errorf("decoding response body: %w", err)
	}
	return resp.Body, resp.Base64Encoded, nil
}

// SetExtraHeaders sets extra HTTP headers to be sent with every request.
func SetExtraHeaders(ctx context.Context, client *webkit.Client, headers map[string]string) error {
	_, err := client.Send(ctx, "Network.setExtraHTTPHeaders", map[string]interface{}{
		"headers": headers,
	})
	return err
}

// InterceptedRequest holds an intercepted request waiting for a continue/response decision.
type InterceptedRequest struct {
	RequestID string                `json:"request_id"`
	Stage     string                `json:"stage"`
	Request   webkit.NetworkRequest `json:"request"`
}

// InterceptionCollector collects Network.requestIntercepted events.
type InterceptionCollector struct {
	mu      sync.Mutex
	pending map[string]*InterceptedRequest
	started bool
}

// NewInterceptionCollector creates a new InterceptionCollector.
func NewInterceptionCollector() *InterceptionCollector {
	return &InterceptionCollector{
		pending: make(map[string]*InterceptedRequest),
	}
}

// Start enables request interception, adds an interception rule, and registers the event handler.
// urlPattern is a URL pattern to intercept (empty string = all requests).
// stage is "request" or "response" (empty defaults to "request").
// isRegex controls whether urlPattern is treated as a regex.
func (ic *InterceptionCollector) Start(ctx context.Context, client *webkit.Client, urlPattern, stage string, isRegex bool) error {
	ic.mu.Lock()
	if ic.started {
		ic.mu.Unlock()
		return nil
	}
	ic.started = true
	ic.mu.Unlock()

	// Network domain must be enabled for interception events to be dispatched.
	_, _ = client.Send(ctx, "Network.enable", nil)

	_, err := client.Send(ctx, "Network.setInterceptionEnabled", map[string]interface{}{
		"enabled": true,
	})
	if err != nil {
		// "Interception already enabled" is non-fatal — a previous session may have
		// left it on. We still need to add rules and register the event handler.
		if !isAlreadyEnabledErr(err) {
			ic.mu.Lock()
			ic.started = false
			ic.mu.Unlock()
			return err
		}
	}

	if stage == "" {
		stage = "request"
	}

	// Register an interception rule — without this, no requestIntercepted events fire.
	_, err = client.Send(ctx, "Network.addInterception", map[string]interface{}{
		"url":     urlPattern,
		"stage":   stage,
		"isRegex": isRegex,
	})
	if err != nil {
		// Roll back — disable interception if we can't add a rule.
		_, _ = client.Send(ctx, "Network.setInterceptionEnabled", map[string]interface{}{
			"enabled": false,
		})
		ic.mu.Lock()
		ic.started = false
		ic.mu.Unlock()
		return fmt.Errorf("adding interception rule: %w", err)
	}

	client.OnEvent("Network.requestIntercepted", func(method string, params json.RawMessage) {
		var evt struct {
			RequestID string                `json:"requestId"`
			Request   webkit.NetworkRequest `json:"request"`
		}
		if err := json.Unmarshal(params, &evt); err != nil {
			return
		}
		ic.mu.Lock()
		ic.pending[evt.RequestID] = &InterceptedRequest{
			RequestID: evt.RequestID,
			Stage:     "request",
			Request:   evt.Request,
		}
		ic.mu.Unlock()
	})

	client.OnEvent("Network.responseIntercepted", func(method string, params json.RawMessage) {
		var evt struct {
			RequestID string                `json:"requestId"`
			Request   webkit.NetworkRequest `json:"request"`
		}
		if err := json.Unmarshal(params, &evt); err != nil {
			return
		}
		ic.mu.Lock()
		ic.pending[evt.RequestID] = &InterceptedRequest{
			RequestID: evt.RequestID,
			Stage:     "response",
			Request:   evt.Request,
		}
		ic.mu.Unlock()
	})

	return nil
}

// Stop disables request interception and removes all interception rules.
func (ic *InterceptionCollector) Stop(ctx context.Context, client *webkit.Client) error {
	ic.mu.Lock()
	ic.started = false
	ic.pending = make(map[string]*InterceptedRequest)
	ic.mu.Unlock()
	// removeInterception is best-effort — the disable call below will clear everything anyway.
	_, _ = client.Send(ctx, "Network.removeInterception", map[string]interface{}{
		"url":   "",
		"stage": "request",
	})
	_, err := client.Send(ctx, "Network.setInterceptionEnabled", map[string]interface{}{
		"enabled": false,
	})
	return err
}

// GetPending returns all pending intercepted requests.
func (ic *InterceptionCollector) GetPending() []InterceptedRequest {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	result := make([]InterceptedRequest, 0, len(ic.pending))
	for _, req := range ic.pending {
		result = append(result, *req)
	}
	return result
}

// RemovePending removes a request from the pending list (after continue/response).
func (ic *InterceptionCollector) RemovePending(requestID string) {
	ic.mu.Lock()
	delete(ic.pending, requestID)
	ic.mu.Unlock()
}

// SetRequestInterception enables or disables request interception directly.
// Prefer using InterceptionCollector for the full interception workflow.
func SetRequestInterception(ctx context.Context, client *webkit.Client, enabled bool) error {
	_, err := client.Send(ctx, "Network.setInterceptionEnabled", map[string]interface{}{
		"enabled": enabled,
	})
	return err
}

// InterceptContinue continues an intercepted request without modification.
func InterceptContinue(ctx context.Context, client *webkit.Client, requestID, stage string) error {
	if stage == "" {
		stage = "request"
	}
	_, err := client.Send(ctx, "Network.interceptContinue", map[string]string{
		"requestId": requestID,
		"stage":     stage,
	})
	return err
}

// InterceptWithResponse provides a custom response for an intercepted request.
// For request-stage interceptions, uses Network.interceptRequestWithResponse (synthetic response).
// For response-stage interceptions, uses Network.interceptWithResponse (modify received response).
func InterceptWithResponse(ctx context.Context, client *webkit.Client, requestID string, stage string, statusCode int, headers map[string]string, content string, base64Encoded bool) error {
	if stage == "" {
		stage = "request"
	}
	if headers == nil {
		headers = map[string]string{}
	}

	if stage == "request" {
		// interceptRequestWithResponse: synthetic response at request stage (skip network).
		_, err := client.Send(ctx, "Network.interceptRequestWithResponse", map[string]interface{}{
			"requestId":     requestID,
			"status":        statusCode,
			"statusText":    httpStatusText(statusCode),
			"mimeType":      mimeTypeFromHeaders(headers),
			"content":       content,
			"base64Encoded": base64Encoded,
			"headers":       headers,
		})
		return err
	}

	// interceptWithResponse: modify response at response stage.
	_, err := client.Send(ctx, "Network.interceptWithResponse", map[string]interface{}{
		"requestId":     requestID,
		"stage":         stage,
		"statusCode":    statusCode,
		"headers":       headers,
		"content":       content,
		"base64Encoded": base64Encoded,
	})
	return err
}

// SetEmulatedConditions configures network emulation with bandwidth and latency constraints.
func SetEmulatedConditions(ctx context.Context, client *webkit.Client, bytesPerSecondLimit int, latencyMs float64) error {
	_, err := client.Send(ctx, "Network.setEmulatedConditions", map[string]interface{}{
		"bytesPerSecondLimit": bytesPerSecondLimit,
		"latencyMs":           latencyMs,
	})
	return err
}

// SetResourceCachingDisabled enables or disables the resource cache.
func SetResourceCachingDisabled(ctx context.Context, client *webkit.Client, disabled bool) error {
	_, err := client.Send(ctx, "Network.setResourceCachingDisabled", map[string]interface{}{
		"disabled": disabled,
	})
	return err
}
