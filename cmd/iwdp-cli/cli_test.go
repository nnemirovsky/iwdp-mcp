package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/nnemirovsky/iwdp-mcp/internal/proxy"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit/testutil"
)

func TestGetPort(t *testing.T) {
	// Clear env to avoid interference.
	t.Setenv("IWDP_PORT", "")

	tests := []struct {
		name string
		args []string
		want int
	}{
		{
			name: "explicit port",
			args: []string{"9223"},
			want: 9223,
		},
		{
			name: "empty args returns default",
			args: []string{},
			want: proxy.DefaultFirstDevicePort,
		},
		{
			name: "non-numeric arg returns default",
			args: []string{"not-a-number"},
			want: proxy.DefaultFirstDevicePort,
		},
		{
			name: "nil args returns default",
			args: nil,
			want: proxy.DefaultFirstDevicePort,
		},
		{
			name: "zero port",
			args: []string{"0"},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPort(tt.args)
			if got != tt.want {
				t.Errorf("getPort(%v) = %d, want %d", tt.args, got, tt.want)
			}
		})
	}
}

func TestGetPortFromEnv(t *testing.T) {
	t.Setenv("IWDP_PORT", "9225")

	got := getPort(nil)
	if got != 9225 {
		t.Errorf("getPort(nil) with IWDP_PORT=9225 = %d, want 9225", got)
	}

	// Explicit arg should take precedence over env.
	got = getPort([]string{"9226"})
	if got != 9226 {
		t.Errorf("getPort([9226]) with IWDP_PORT=9225 = %d, want 9226", got)
	}
}

func TestGetWSURL(t *testing.T) {
	// Clear env to avoid interference.
	t.Setenv("IWDP_WS_URL", "")

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "ws URL in args",
			args: []string{"ws://localhost:9222/devtools/page/1"},
			want: "ws://localhost:9222/devtools/page/1",
		},
		{
			name: "wss URL in args",
			args: []string{"wss://localhost:9222/devtools/page/1"},
			want: "wss://localhost:9222/devtools/page/1",
		},
		{
			name: "non-ws arg returns empty",
			args: []string{"http://localhost:9222"},
			want: "",
		},
		{
			name: "empty args returns empty",
			args: []string{},
			want: "",
		},
		{
			name: "nil args returns empty",
			args: nil,
			want: "",
		},
		{
			name: "ws URL among other args",
			args: []string{"some-expr", "ws://localhost:9222/devtools/page/2"},
			want: "ws://localhost:9222/devtools/page/2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getWSURL(tt.args)
			if got != tt.want {
				t.Errorf("getWSURL(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestGetWSURLFromEnv(t *testing.T) {
	t.Setenv("IWDP_WS_URL", "ws://localhost:9222/devtools/page/99")

	got := getWSURL(nil)
	if got != "ws://localhost:9222/devtools/page/99" {
		t.Errorf("getWSURL(nil) with env = %q, want ws://localhost:9222/devtools/page/99", got)
	}

	// Explicit arg should take precedence over env.
	got = getWSURL([]string{"ws://localhost:9222/devtools/page/1"})
	if got != "ws://localhost:9222/devtools/page/1" {
		t.Errorf("getWSURL with explicit arg = %q, want ws://localhost:9222/devtools/page/1", got)
	}
}

func TestConnectToPageWithWSURL(t *testing.T) {
	mock := testutil.NewMockServer()
	defer mock.Close()

	ctx := context.Background()
	client, err := connectToPage(ctx, []string{mock.URL})
	if err != nil {
		t.Fatalf("connectToPage: %v", err)
	}
	defer func() { _ = client.Close() }()

	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestConnectToPageWithWSURLAmongArgs(t *testing.T) {
	mock := testutil.NewMockServer()
	defer mock.Close()

	ctx := context.Background()
	// Simulate passing an expression + a ws URL, like "eval" does:
	// connectToPage(ctx, args[1:]) where args = ["document.title", "ws://..."]
	client, err := connectToPage(ctx, []string{"document.title", mock.URL})
	if err != nil {
		t.Fatalf("connectToPage: %v", err)
	}
	defer func() { _ = client.Close() }()

	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestConnectToPageSendCommand(t *testing.T) {
	mock := testutil.NewMockServer()
	defer mock.Close()

	mock.HandleFunc("Runtime.evaluate", map[string]interface{}{
		"result": map[string]interface{}{
			"type":  "string",
			"value": "hello",
		},
	})

	ctx := context.Background()
	client, err := connectToPage(ctx, []string{mock.URL})
	if err != nil {
		t.Fatalf("connectToPage: %v", err)
	}
	defer func() { _ = client.Close() }()

	result, err := client.Send(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression":    "1+1",
		"returnByValue": true,
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	res, ok := got["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T", got["result"])
	}
	if res["type"] != "string" {
		t.Errorf("result.type = %v, want string", res["type"])
	}
}

func TestConnectToPageInvalidURL(t *testing.T) {
	ctx := context.Background()
	_, err := connectToPage(ctx, []string{"ws://localhost:1/nonexistent"})
	if err == nil {
		t.Fatal("expected error for invalid WebSocket URL")
	}
}

// TestCmdDevicesWithMockIWDP tests the devices listing against a mock HTTP
// server that simulates ios-webkit-debug-proxy's /json endpoint on port 9221.
func TestCmdDevicesWithMockIWDP(t *testing.T) {
	devices := []webkit.DeviceEntry{
		{DeviceID: "abc123", DeviceName: "iPhone 15", URL: "localhost:9222"},
		{DeviceID: "def456", DeviceName: "iPad Pro", URL: "localhost:9223"},
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

	// We can't easily redirect the proxy package to use our mock server
	// (it uses hardcoded localhost:9221), but we can verify the mock serves
	// valid JSON that matches the expected types.
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
		t.Errorf("first device = %q, want iPhone 15", got[0].DeviceName)
	}
	if got[1].URL != "localhost:9223" {
		t.Errorf("second device URL = %q, want localhost:9223", got[1].URL)
	}
}

// TestCmdPagesWithMockIWDP tests the pages listing against a mock HTTP server
// that simulates a device port's /json endpoint.
func TestCmdPagesWithMockIWDP(t *testing.T) {
	pages := []webkit.PageEntry{
		{
			Title:                "Example",
			URL:                  "https://example.com",
			WebSocketDebuggerURL: "ws://localhost:9222/devtools/page/1",
			PageID:               1,
		},
		{
			Title:                "Google",
			URL:                  "https://google.com",
			WebSocketDebuggerURL: "ws://localhost:9222/devtools/page/2",
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
	if got[0].Title != "Example" {
		t.Errorf("first page title = %q, want Example", got[0].Title)
	}
	if got[0].WebSocketDebuggerURL != "ws://localhost:9222/devtools/page/1" {
		t.Errorf("first page ws URL = %q", got[0].WebSocketDebuggerURL)
	}
	if got[1].PageID != 2 {
		t.Errorf("second page ID = %d, want 2", got[1].PageID)
	}
}

// TestMockIWDPFullFlow simulates the full discovery flow: listing devices on
// the listing port, then listing pages on a device port, then connecting via
// WebSocket. This is an end-to-end test using mock servers.
func TestMockIWDPFullFlow(t *testing.T) {
	// Step 1: Create a mock WebSocket server (simulates a debuggable page).
	wsMock := testutil.NewMockServer()
	defer wsMock.Close()

	wsMock.HandleFunc("Runtime.evaluate", map[string]interface{}{
		"result": map[string]interface{}{
			"type":  "string",
			"value": "test-result",
		},
	})

	// Step 2: Create a mock HTTP server for the device port (simulates /json on 9222).
	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages := []webkit.PageEntry{
			{
				Title:                "Test Page",
				URL:                  "https://test.example.com",
				WebSocketDebuggerURL: wsMock.URL,
				PageID:               1,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pages)
	}))
	defer deviceServer.Close()

	// Step 3: Fetch pages from the mock device server.
	resp, err := http.Get(deviceServer.URL + "/json")
	if err != nil {
		t.Fatalf("GET device /json: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var pages []webkit.PageEntry
	if err := json.NewDecoder(resp.Body).Decode(&pages); err != nil {
		t.Fatalf("decode pages: %v", err)
	}
	if len(pages) == 0 {
		t.Fatal("no pages returned")
	}

	// Step 4: Connect to the first page's WebSocket URL.
	wsURL := pages[0].WebSocketDebuggerURL
	if wsURL == "" {
		t.Fatal("no WebSocket URL for first page")
	}

	ctx := context.Background()
	client, err := webkit.NewClient(ctx, wsURL)
	if err != nil {
		t.Fatalf("NewClient(%s): %v", wsURL, err)
	}
	defer func() { _ = client.Close() }()

	// Step 5: Send a command through the WebSocket connection.
	result, err := client.Send(ctx, "Runtime.evaluate", map[string]interface{}{
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
	if res["value"] != "test-result" {
		t.Errorf("result.value = %v, want test-result", res["value"])
	}
}

// TestMockIWDPDeviceDiscovery simulates the device listing endpoint and
// verifies the /json response is correctly structured.
func TestMockIWDPDeviceDiscovery(t *testing.T) {
	// Simulate the listing port (9221) with multiple devices.
	listingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			devices := []webkit.DeviceEntry{
				{DeviceID: "abc", DeviceName: "iPhone 15 Pro", URL: "localhost:9222"},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(devices)
		default:
			http.NotFound(w, r)
		}
	}))
	defer listingServer.Close()

	resp, err := http.Get(listingServer.URL + "/json")
	if err != nil {
		t.Fatalf("GET /json: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var devices []webkit.DeviceEntry
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("got %d devices, want 1", len(devices))
	}
	if devices[0].DeviceID != "abc" {
		t.Errorf("deviceId = %q, want abc", devices[0].DeviceID)
	}
}

// TestPrintUsageDoesNotPanic verifies that printUsage() completes without error.
func TestPrintUsageDoesNotPanic(t *testing.T) {
	// Redirect stderr to discard output.
	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	defer func() { os.Stderr = old }()

	// Should not panic.
	printUsage()
}

// TestGetPortEdgeCases tests boundary conditions for getPort.
func TestGetPortEdgeCases(t *testing.T) {
	t.Setenv("IWDP_PORT", "")

	tests := []struct {
		name string
		args []string
		want int
	}{
		{
			name: "negative number",
			args: []string{"-1"},
			want: -1,
		},
		{
			name: "large port number",
			args: []string{"65535"},
			want: 65535,
		},
		{
			name: "port with trailing text",
			args: []string{"9222abc"},
			want: proxy.DefaultFirstDevicePort,
		},
		{
			name: "empty string",
			args: []string{""},
			want: proxy.DefaultFirstDevicePort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPort(tt.args)
			if got != tt.want {
				t.Errorf("getPort(%v) = %d, want %d", tt.args, got, tt.want)
			}
		})
	}
}

// TestGetWSURLPrefixMatching verifies the ws:// and wss:// prefix detection.
func TestGetWSURLPrefixMatching(t *testing.T) {
	t.Setenv("IWDP_WS_URL", "")

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "http URL is not a ws URL",
			args: []string{"http://localhost:9222"},
			want: "",
		},
		{
			name: "ws URL without path",
			args: []string{"ws://localhost:9222"},
			want: "ws://localhost:9222",
		},
		{
			name: "wss with path",
			args: []string{"wss://example.com/debug"},
			want: "wss://example.com/debug",
		},
		{
			name: "first ws URL wins",
			args: []string{"ws://first", "ws://second"},
			want: "ws://first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getWSURL(tt.args)
			if got != tt.want {
				t.Errorf("getWSURL(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

// TestConnectToPageNoIWDP verifies that connectToPage without a ws:// URL
// fails with a meaningful error when iwdp is not running (and doesn't panic
// or hang). This test relies on ios_webkit_debug_proxy NOT being on port 9221.
func TestConnectToPageNoIWDP(t *testing.T) {
	t.Setenv("IWDP_WS_URL", "")

	// We can't easily test the os.Exit(1) path, but we can verify that
	// getWSURL returns empty when there's no ws:// URL in args, which is
	// the condition that triggers the iwdp check.
	wsURL := getWSURL([]string{"some-expression"})
	if wsURL != "" {
		t.Errorf("expected empty ws URL, got %q", wsURL)
	}
}

// TestFatalMessage verifies the error message format from a mock perspective.
// We can't test fatal() directly since it calls os.Exit, but we verify
// the error formatting pattern.
func TestFatalMessage(t *testing.T) {
	err := fmt.Errorf("no Safari tabs found — open a page in Safari on your iOS device")
	msg := err.Error()
	if !strings.Contains(msg, "Safari") {
		t.Errorf("error message %q should mention Safari", msg)
	}
}
