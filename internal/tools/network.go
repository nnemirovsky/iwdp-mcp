package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

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
			Request webkit.NetworkRequest `json:"request"`
		}
		if err := json.Unmarshal(params, &evt); err != nil {
			return
		}
		m.mu.Lock()
		if len(m.requests) >= maxCollectorEntries {
			m.mu.Unlock()
			return
		}
		m.requests[evt.Request.RequestID] = &CapturedRequest{
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

// SetRequestInterception enables or disables request interception.
func SetRequestInterception(ctx context.Context, client *webkit.Client, enabled bool) error {
	_, err := client.Send(ctx, "Network.setInterceptionEnabled", map[string]interface{}{
		"enabled": enabled,
	})
	return err
}

// InterceptContinue continues an intercepted request without modification.
func InterceptContinue(ctx context.Context, client *webkit.Client, requestID string) error {
	_, err := client.Send(ctx, "Network.interceptContinue", map[string]string{
		"requestId": requestID,
	})
	return err
}

// InterceptWithResponse provides a custom response for an intercepted request.
func InterceptWithResponse(ctx context.Context, client *webkit.Client, requestID string, statusCode int, headers map[string]string, body string) error {
	_, err := client.Send(ctx, "Network.interceptWithResponse", map[string]interface{}{
		"requestId":  requestID,
		"statusCode": statusCode,
		"headers":    headers,
		"body":       body,
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
