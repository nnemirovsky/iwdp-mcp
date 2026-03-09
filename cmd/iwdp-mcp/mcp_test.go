package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit/testutil"
)

func TestServerCreation(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "iwdp-mcp",
		Version: "0.1.0",
	}, nil)
	if server == nil {
		t.Fatal("server is nil")
	}
}

func TestServerCreationWithRegisterTools(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "iwdp-mcp",
		Version: "0.1.0",
	}, nil)
	// registerTools should not panic.
	registerTools(server)
	if server == nil {
		t.Fatal("server is nil after registerTools")
	}
}

func TestGetClientNoSession(t *testing.T) {
	// Reset global session state before the test.
	sess.mu.Lock()
	oldClient := sess.client
	sess.client = nil
	sess.mu.Unlock()
	defer func() {
		sess.mu.Lock()
		sess.client = oldClient
		sess.mu.Unlock()
	}()

	ctx := context.Background()
	_, err := getClient(ctx)
	if err == nil {
		t.Fatal("expected error when no page selected")
	}
	if err.Error() != "no page selected — use select_page first" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestSessionStateNilClient(t *testing.T) {
	// Reset session state.
	sess.mu.Lock()
	oldClient := sess.client
	oldNM := sess.networkMonitor
	oldCC := sess.consoleCollector
	oldTC := sess.timelineCollector
	sess.client = nil
	sess.networkMonitor = nil
	sess.consoleCollector = nil
	sess.timelineCollector = nil
	sess.mu.Unlock()
	defer func() {
		sess.mu.Lock()
		sess.client = oldClient
		sess.networkMonitor = oldNM
		sess.consoleCollector = oldCC
		sess.timelineCollector = oldTC
		sess.mu.Unlock()
	}()

	ctx := context.Background()
	_, err := getClient(ctx)
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestSessionStateWithClient(t *testing.T) {
	mock := testutil.NewMockServer()
	defer mock.Close()

	ctx := context.Background()
	client, err := webkit.NewClient(ctx, mock.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Save and restore global session state.
	sess.mu.Lock()
	oldClient := sess.client
	sess.client = client
	sess.mu.Unlock()
	defer func() {
		sess.mu.Lock()
		sess.client = oldClient
		sess.mu.Unlock()
		_ = client.Close()
	}()

	got, err := getClient(ctx)
	if err != nil {
		t.Fatalf("getClient: %v", err)
	}
	if got != client {
		t.Error("getClient returned a different client than expected")
	}
}

func TestSessionStateClientSwitch(t *testing.T) {
	mock1 := testutil.NewMockServer()
	defer mock1.Close()
	mock2 := testutil.NewMockServer()
	defer mock2.Close()

	ctx := context.Background()

	client1, err := webkit.NewClient(ctx, mock1.URL)
	if err != nil {
		t.Fatalf("NewClient 1: %v", err)
	}

	client2, err := webkit.NewClient(ctx, mock2.URL)
	if err != nil {
		t.Fatalf("NewClient 2: %v", err)
	}

	// Save and restore global session state.
	sess.mu.Lock()
	oldClient := sess.client
	oldNM := sess.networkMonitor
	oldCC := sess.consoleCollector
	oldTC := sess.timelineCollector
	sess.mu.Unlock()
	defer func() {
		sess.mu.Lock()
		sess.client = oldClient
		sess.networkMonitor = oldNM
		sess.consoleCollector = oldCC
		sess.timelineCollector = oldTC
		sess.mu.Unlock()
		_ = client1.Close()
		_ = client2.Close()
	}()

	// Set client1.
	sess.mu.Lock()
	sess.client = client1
	sess.mu.Unlock()

	got, err := getClient(ctx)
	if err != nil {
		t.Fatalf("getClient after client1: %v", err)
	}
	if got != client1 {
		t.Error("expected client1")
	}

	// Switch to client2.
	sess.mu.Lock()
	sess.client = client2
	sess.networkMonitor = nil
	sess.consoleCollector = nil
	sess.timelineCollector = nil
	sess.mu.Unlock()

	got, err = getClient(ctx)
	if err != nil {
		t.Fatalf("getClient after client2: %v", err)
	}
	if got != client2 {
		t.Error("expected client2")
	}
}

// TestMockIWDPDeviceListing tests that a mock IWDP listing endpoint returns
// device entries that match the expected structure.
func TestMockIWDPDeviceListing(t *testing.T) {
	devices := []webkit.DeviceEntry{
		{DeviceID: "abc123", DeviceName: "iPhone 15", URL: "localhost:9222"},
		{DeviceID: "def456", DeviceName: "iPad Air", URL: "localhost:9223"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(devices)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/json")
	if err != nil {
		t.Fatalf("GET /json: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var got []webkit.DeviceEntry
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d devices, want 2", len(got))
	}
	if got[0].DeviceName != "iPhone 15" {
		t.Errorf("device 0 name = %q, want iPhone 15", got[0].DeviceName)
	}
	if got[1].URL != "localhost:9223" {
		t.Errorf("device 1 URL = %q, want localhost:9223", got[1].URL)
	}
}

// TestMockIWDPPageListing tests that a mock device port endpoint returns
// page entries with WebSocket debugger URLs.
func TestMockIWDPPageListing(t *testing.T) {
	pages := []webkit.PageEntry{
		{
			Title:                "Example Page",
			URL:                  "https://example.com",
			WebSocketDebuggerURL: "ws://localhost:9222/devtools/page/1",
			PageID:               1,
		},
		{
			Title:                "Another Page",
			URL:                  "https://other.com",
			WebSocketDebuggerURL: "",
			PageID:               2,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pages)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/json")
	if err != nil {
		t.Fatalf("GET /json: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var got []webkit.PageEntry
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d pages, want 2", len(got))
	}
	if got[0].WebSocketDebuggerURL != "ws://localhost:9222/devtools/page/1" {
		t.Errorf("page 0 ws URL = %q", got[0].WebSocketDebuggerURL)
	}
	if got[1].WebSocketDebuggerURL != "" {
		t.Errorf("page 1 ws URL should be empty, got %q", got[1].WebSocketDebuggerURL)
	}
}

// TestMockIWDPFullFlow is an end-to-end test that simulates the complete flow:
// device discovery -> page listing -> WebSocket connection -> command execution.
func TestMockIWDPFullFlow(t *testing.T) {
	// Step 1: Create a mock WebSocket server simulating a debuggable page.
	wsMock := testutil.NewMockServer()
	defer wsMock.Close()

	wsMock.HandleFunc("Runtime.evaluate", map[string]interface{}{
		"result": map[string]interface{}{
			"type":  "string",
			"value": "mcp-test-result",
		},
	})

	// Step 2: Mock the device port's /json endpoint pointing to the ws server.
	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages := []webkit.PageEntry{
			{
				Title:                "MCP Test Page",
				URL:                  "https://test.example.com",
				WebSocketDebuggerURL: wsMock.URL,
				PageID:               1,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pages)
	}))
	defer deviceServer.Close()

	// Step 3: Fetch pages.
	resp, err := http.Get(deviceServer.URL + "/json")
	if err != nil {
		t.Fatalf("GET /json: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var pages []webkit.PageEntry
	if err := json.NewDecoder(resp.Body).Decode(&pages); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(pages) == 0 {
		t.Fatal("no pages")
	}

	// Step 4: Connect via WebSocket and set as active client.
	ctx := context.Background()
	client, err := webkit.NewClient(ctx, pages[0].WebSocketDebuggerURL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Save and restore global session state.
	sess.mu.Lock()
	oldClient := sess.client
	sess.client = client
	sess.mu.Unlock()
	defer func() {
		sess.mu.Lock()
		sess.client = oldClient
		sess.mu.Unlock()
	}()

	// Step 5: Verify getClient returns the connected client.
	got, err := getClient(ctx)
	if err != nil {
		t.Fatalf("getClient: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil client from getClient")
	}

	// Step 6: Send a command through the active client.
	result, err := got.Send(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression":    "'hello'",
		"returnByValue": true,
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	var evalResult map[string]interface{}
	if err := json.Unmarshal(result, &evalResult); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	res, ok := evalResult["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T", evalResult["result"])
	}
	if res["value"] != "mcp-test-result" {
		t.Errorf("result.value = %v, want mcp-test-result", res["value"])
	}
}

// TestOKHelper verifies the ok() helper function.
func TestOKHelper(t *testing.T) {
	result := ok()
	if !result.OK {
		t.Error("ok() should return OKOutput{OK: true}")
	}
}

// TestEmptyDeviceList tests handling of an empty device list from the proxy.
func TestEmptyDeviceList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]webkit.DeviceEntry{})
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/json")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var devices []webkit.DeviceEntry
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("expected empty device list, got %d", len(devices))
	}
}

// TestEmptyPageList tests handling of an empty page list from a device port.
func TestEmptyPageList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]webkit.PageEntry{})
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/json")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var pages []webkit.PageEntry
	if err := json.NewDecoder(resp.Body).Decode(&pages); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(pages) != 0 {
		t.Errorf("expected empty page list, got %d", len(pages))
	}
}

// TestWebSocketConnectionAndCommand tests connecting to a mock WebSocket
// server and sending multiple WebKit protocol commands.
func TestWebSocketConnectionAndCommand(t *testing.T) {
	mock := testutil.NewMockServer()
	defer mock.Close()

	mock.HandleFunc("Page.navigate", map[string]interface{}{
		"frameId": "main",
	})
	mock.HandleFunc("Page.captureScreenshot", map[string]interface{}{
		"data": "iVBORw0KGgo=",
	})
	mock.HandleFunc("Runtime.evaluate", map[string]interface{}{
		"result": map[string]interface{}{
			"type":  "number",
			"value": 42,
		},
	})

	ctx := context.Background()
	client, err := webkit.NewClient(ctx, mock.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Test navigate.
	result, err := client.Send(ctx, "Page.navigate", map[string]string{"url": "https://example.com"})
	if err != nil {
		t.Fatalf("navigate: %v", err)
	}
	var navResult map[string]interface{}
	_ = json.Unmarshal(result, &navResult)
	if navResult["frameId"] != "main" {
		t.Errorf("navigate frameId = %v, want main", navResult["frameId"])
	}

	// Test screenshot.
	result, err = client.Send(ctx, "Page.captureScreenshot", nil)
	if err != nil {
		t.Fatalf("screenshot: %v", err)
	}
	var ssResult map[string]interface{}
	_ = json.Unmarshal(result, &ssResult)
	if ssResult["data"] == nil {
		t.Error("screenshot data is nil")
	}

	// Test evaluate.
	result, err = client.Send(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression":    "21 * 2",
		"returnByValue": true,
	})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	var evalResult map[string]interface{}
	_ = json.Unmarshal(result, &evalResult)
	res, ok := evalResult["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map")
	}
	if res["value"] != float64(42) {
		t.Errorf("evaluate value = %v, want 42", res["value"])
	}
}
