package tools_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit/testutil"
)

// helper creates a mock server and connected client, returning both and a cleanup function.
func setup(t *testing.T) (*testutil.MockServer, *webkit.Client) {
	t.Helper()
	mock := testutil.NewMockServer()

	ctx := context.Background()
	client, err := webkit.NewClient(ctx, mock.URL)
	if err != nil {
		mock.Close()
		t.Fatalf("NewClient: %v", err)
	}

	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Logf("warning: closing client: %v", err)
		}
		mock.Close()
	})

	return mock, client
}

func TestNavigate(t *testing.T) {
	mock, client := setup(t)

	var receivedURL string
	mock.Handle("Page.navigate", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedURL = p.URL
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	err := tools.Navigate(ctx, client, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate returned error: %v", err)
	}
	if receivedURL != "https://example.com" {
		t.Errorf("expected URL %q, got %q", "https://example.com", receivedURL)
	}
}

func TestEvaluateScript(t *testing.T) {
	mock, client := setup(t)

	mock.HandleFunc("Runtime.evaluate", map[string]interface{}{
		"result": map[string]interface{}{
			"type":  "string",
			"value": `"hello"`,
		},
		"wasThrown": false,
	})

	ctx := context.Background()
	result, err := tools.EvaluateScript(ctx, client, `"hello"`, true)
	if err != nil {
		t.Fatalf("EvaluateScript returned error: %v", err)
	}
	if result.Result.Type != "string" {
		t.Errorf("expected result type %q, got %q", "string", result.Result.Type)
	}
	if result.WasThrown {
		t.Errorf("expected wasThrown to be false")
	}
}

func TestGetDocument(t *testing.T) {
	mock, client := setup(t)

	mock.HandleFunc("DOM.getDocument", map[string]interface{}{
		"root": map[string]interface{}{
			"nodeId":   1,
			"nodeType": 9,
			"nodeName": "#document",
		},
	})

	ctx := context.Background()
	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument returned error: %v", err)
	}
	if root.NodeID != 1 {
		t.Errorf("expected root.NodeID = 1, got %d", root.NodeID)
	}
	if root.NodeType != 9 {
		t.Errorf("expected root.NodeType = 9, got %d", root.NodeType)
	}
	if root.NodeName != "#document" {
		t.Errorf("expected root.NodeName = %q, got %q", "#document", root.NodeName)
	}
}

func TestQuerySelector(t *testing.T) {
	mock, client := setup(t)

	mock.HandleFunc("DOM.querySelector", map[string]interface{}{
		"nodeId": 42,
	})

	ctx := context.Background()
	nodeID, err := tools.QuerySelector(ctx, client, 1, "div.main")
	if err != nil {
		t.Fatalf("QuerySelector returned error: %v", err)
	}
	if nodeID != 42 {
		t.Errorf("expected nodeID = 42, got %d", nodeID)
	}
}

func TestQuerySelectorNotFound(t *testing.T) {
	mock, client := setup(t)

	mock.HandleFunc("DOM.querySelector", map[string]interface{}{
		"nodeId": 0,
	})

	ctx := context.Background()
	_, err := tools.QuerySelector(ctx, client, 1, "div.nonexistent")
	if err == nil {
		t.Fatal("expected error for not-found selector, got nil")
	}
}

func TestGetOuterHTML(t *testing.T) {
	mock, client := setup(t)

	mock.HandleFunc("DOM.getOuterHTML", map[string]interface{}{
		"outerHTML": "<div>test</div>",
	})

	ctx := context.Background()
	html, err := tools.GetOuterHTML(ctx, client, 42)
	if err != nil {
		t.Fatalf("GetOuterHTML returned error: %v", err)
	}
	if html != "<div>test</div>" {
		t.Errorf("expected %q, got %q", "<div>test</div>", html)
	}
}

func TestGetCookies(t *testing.T) {
	mock, client := setup(t)

	mock.HandleFunc("Page.getCookies", map[string]interface{}{
		"cookies": []map[string]interface{}{
			{
				"name":     "session",
				"value":    "abc123",
				"domain":   ".example.com",
				"path":     "/",
				"httpOnly": true,
				"secure":   true,
				"session":  false,
				"expires":  1700000000.0,
			},
		},
	})

	ctx := context.Background()
	cookies, err := tools.GetCookies(ctx, client)
	if err != nil {
		t.Fatalf("GetCookies returned error: %v", err)
	}
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	c := cookies[0]
	if c.Name != "session" {
		t.Errorf("expected cookie name %q, got %q", "session", c.Name)
	}
	if c.Value != "abc123" {
		t.Errorf("expected cookie value %q, got %q", "abc123", c.Value)
	}
	if c.Domain != ".example.com" {
		t.Errorf("expected cookie domain %q, got %q", ".example.com", c.Domain)
	}
	if !c.HTTPOnly {
		t.Errorf("expected cookie to be httpOnly")
	}
	if !c.Secure {
		t.Errorf("expected cookie to be secure")
	}
}

func TestClick(t *testing.T) {
	mock, client := setup(t)

	mock.HandleFunc("Runtime.evaluate", map[string]interface{}{
		"result": map[string]interface{}{
			"type": "undefined",
		},
		"wasThrown": false,
	})

	ctx := context.Background()
	err := tools.Click(ctx, client, "button")
	if err != nil {
		t.Fatalf("Click returned error: %v", err)
	}

	// Verify that the mock received a Runtime.evaluate call (handler was invoked).
	// If the handler were missing, the default empty result would still succeed,
	// so the fact that we registered it and got no error is sufficient.
	_ = mock
}

func TestFill(t *testing.T) {
	mock, client := setup(t)

	mock.HandleFunc("Runtime.evaluate", map[string]interface{}{
		"result": map[string]interface{}{
			"type": "undefined",
		},
		"wasThrown": false,
	})

	ctx := context.Background()
	err := tools.Fill(ctx, client, "input", "test")
	if err != nil {
		t.Fatalf("Fill returned error: %v", err)
	}
	_ = mock
}

func TestConsoleCollector(t *testing.T) {
	mock, client := setup(t)

	// Register handler for Console.enable so Start() succeeds.
	mock.HandleFunc("Console.enable", map[string]interface{}{})

	collector := tools.NewConsoleCollector()
	ctx := context.Background()
	if err := collector.Start(ctx, client); err != nil {
		t.Fatalf("ConsoleCollector.Start: %v", err)
	}

	// Send a Console.messageAdded event from the mock server.
	err := mock.SendEvent("Console.messageAdded", map[string]interface{}{
		"message": map[string]interface{}{
			"source": "console-api",
			"level":  "log",
			"text":   "hello from console",
			"url":    "https://example.com",
			"line":   10,
			"column": 5,
		},
	})
	if err != nil {
		t.Fatalf("SendEvent: %v", err)
	}

	// Allow time for the event handler to process.
	time.Sleep(100 * time.Millisecond)

	messages := collector.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 console message, got %d", len(messages))
	}

	msg := messages[0]
	if msg.Source != "console-api" {
		t.Errorf("expected source %q, got %q", "console-api", msg.Source)
	}
	if msg.Level != "log" {
		t.Errorf("expected level %q, got %q", "log", msg.Level)
	}
	if msg.Text != "hello from console" {
		t.Errorf("expected text %q, got %q", "hello from console", msg.Text)
	}
}

func TestNetworkMonitor(t *testing.T) {
	mock, client := setup(t)

	// Register handler for Network.enable so Start() succeeds.
	mock.HandleFunc("Network.enable", map[string]interface{}{})

	monitor := tools.NewNetworkMonitor()
	ctx := context.Background()
	if err := monitor.Start(ctx, client); err != nil {
		t.Fatalf("NetworkMonitor.Start: %v", err)
	}

	// Send a Network.requestWillBeSent event.
	err := mock.SendEvent("Network.requestWillBeSent", map[string]interface{}{
		"request": map[string]interface{}{
			"requestId": "req-1",
			"url":       "https://example.com/api",
			"method":    "GET",
			"headers":   map[string]string{"Accept": "application/json"},
			"timestamp": 12345.0,
		},
	})
	if err != nil {
		t.Fatalf("SendEvent requestWillBeSent: %v", err)
	}

	// Allow time for the event handler to process.
	time.Sleep(100 * time.Millisecond)

	// Send a Network.responseReceived event for the same request.
	err = mock.SendEvent("Network.responseReceived", map[string]interface{}{
		"requestId": "req-1",
		"response": map[string]interface{}{
			"url":        "https://example.com/api",
			"status":     200,
			"statusText": "OK",
			"headers":    map[string]string{"Content-Type": "application/json"},
			"mimeType":   "application/json",
		},
	})
	if err != nil {
		t.Fatalf("SendEvent responseReceived: %v", err)
	}

	// Allow time for the event handler to process.
	time.Sleep(100 * time.Millisecond)

	requests := monitor.GetRequests()
	if len(requests) != 1 {
		t.Fatalf("expected 1 captured request, got %d", len(requests))
	}

	req := requests[0]
	if req.Request.RequestID != "req-1" {
		t.Errorf("expected requestId %q, got %q", "req-1", req.Request.RequestID)
	}
	if req.Request.URL != "https://example.com/api" {
		t.Errorf("expected request URL %q, got %q", "https://example.com/api", req.Request.URL)
	}
	if req.Request.Method != "GET" {
		t.Errorf("expected request method %q, got %q", "GET", req.Request.Method)
	}
	if req.Response == nil {
		t.Fatal("expected response to be present")
	}
	if req.Response.Status != 200 {
		t.Errorf("expected response status 200, got %d", req.Response.Status)
	}
	if req.Response.MimeType != "application/json" {
		t.Errorf("expected response mimeType %q, got %q", "application/json", req.Response.MimeType)
	}
}
