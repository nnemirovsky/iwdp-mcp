package e2e_test

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	mcpBinaryOnce sync.Once
	mcpBinaryPath string
	mcpBinaryErr  error
)

func buildMCP(t *testing.T) string {
	t.Helper()
	mcpBinaryOnce.Do(func() {
		dir, err := os.MkdirTemp("", "iwdp-e2e-mcp-*")
		if err != nil {
			mcpBinaryErr = err
			return
		}
		trackTempDir(dir)
		bin := filepath.Join(dir, "iwdp-mcp")
		cmd := exec.Command("go", "build", "-o", bin, "../cmd/iwdp-mcp")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			mcpBinaryErr = err
			return
		}
		mcpBinaryPath = bin
	})
	if mcpBinaryErr != nil {
		t.Fatalf("building iwdp-mcp: %v", mcpBinaryErr)
	}
	return mcpBinaryPath
}

// mcpProcess manages a running MCP server subprocess.
type mcpProcess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
}

func startMCP(t *testing.T) *mcpProcess {
	t.Helper()
	bin := buildMCP(t)

	cmd := exec.Command(bin)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("starting MCP server: %v", err)
	}

	p := &mcpProcess{
		cmd:     cmd,
		stdin:   stdin,
		scanner: bufio.NewScanner(stdout),
	}

	t.Cleanup(func() {
		_ = stdin.Close()
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			_ = cmd.Process.Kill()
		}
	})

	return p
}

// send sends a JSON-RPC message as a single NDJSON line.
func (p *mcpProcess) send(t *testing.T, msg interface{}) {
	t.Helper()
	body, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshaling message: %v", err)
	}
	if _, err := io.WriteString(p.stdin, string(body)+"\n"); err != nil {
		t.Fatalf("writing to stdin: %v", err)
	}
}

// recv reads a JSON-RPC response (single NDJSON line), skipping notifications.
func (p *mcpProcess) recv(t *testing.T) map[string]interface{} {
	t.Helper()

	for {
		if !p.scanner.Scan() {
			if err := p.scanner.Err(); err != nil {
				t.Fatalf("reading response: %v", err)
			}
			t.Fatal("no response from MCP server (EOF)")
		}

		line := p.scanner.Text()
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			t.Fatalf("unmarshaling response: %v\nline: %s", err, line)
		}

		// Skip notifications (messages without an id field).
		if _, hasID := result["id"]; hasID {
			return result
		}
	}
}

// initialize sends the initialize handshake and returns the response.
func (p *mcpProcess) initialize(t *testing.T) map[string]interface{} {
	t.Helper()

	p.send(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	})

	resp := p.recv(t)

	// Send initialized notification.
	p.send(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})

	return resp
}

func TestMCP_Initialize(t *testing.T) {
	p := startMCP(t)

	resp := p.initialize(t)

	if resp["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", resp["jsonrpc"])
	}
	if resp["id"] != float64(1) {
		t.Errorf("id = %v, want 1", resp["id"])
	}
	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T", resp["result"])
	}

	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected serverInfo map, got %T", result["serverInfo"])
	}
	if serverInfo["name"] != "iwdp-mcp" {
		t.Errorf("serverInfo.name = %v, want iwdp-mcp", serverInfo["name"])
	}
}

func TestMCP_ListTools(t *testing.T) {
	p := startMCP(t)
	p.initialize(t)

	p.send(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	})

	resp := p.recv(t)

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T", resp["result"])
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatalf("expected tools array, got %T", result["tools"])
	}

	if len(tools) < 50 {
		t.Errorf("expected 50+ tools, got %d", len(tools))
	}

	// Verify key tools are present.
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolMap, ok := tool.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := toolMap["name"].(string)
		toolNames[name] = true
	}

	expectedTools := []string{
		// Core
		"iwdp_status", "restart_iwdp",
		"list_devices", "list_pages", "select_page",
		"navigate", "reload", "evaluate_script", "take_screenshot", "snapshot_node",
		"get_document", "query_selector", "query_selector_all",
		"get_outer_html", "get_attributes", "get_event_listeners",
		"highlight_node", "hide_highlight",
		// Runtime
		"call_function", "get_properties",
		// CSS
		"get_matched_styles", "get_computed_style", "get_inline_styles",
		"set_style_text", "get_all_stylesheets", "get_stylesheet_text",
		"force_pseudo_state",
		// Network
		"network_enable", "network_disable", "list_network_requests",
		"get_response_body", "set_extra_headers",
		"set_request_interception", "list_intercepted_requests",
		"intercept_continue", "intercept_with_response",
		"intercept_continue_all", "intercept_block_all",
		"set_emulated_conditions", "set_resource_caching_disabled",
		// Storage
		"get_cookies", "set_cookie", "delete_cookie",
		"get_local_storage", "set_local_storage_item", "remove_local_storage_item", "clear_local_storage",
		"get_session_storage", "set_session_storage_item", "remove_session_storage_item", "clear_session_storage",
		"list_indexed_databases", "get_indexed_db_data", "clear_indexed_db_store",
		// Console
		"console_enable", "console_disable",
		"get_console_messages", "clear_console", "set_log_level",
		// Debugger
		"debugger_enable", "debugger_disable",
		"set_breakpoint", "remove_breakpoint",
		"pause", "resume", "step_over", "step_into", "step_out",
		"get_script_source", "search_in_content", "evaluate_on_call_frame",
		"set_pause_on_exceptions",
		// DOMDebugger
		"set_dom_breakpoint", "remove_dom_breakpoint",
		"set_event_breakpoint", "remove_event_breakpoint",
		"set_url_breakpoint", "remove_url_breakpoint",
		// Timeline
		"timeline_start", "timeline_stop", "get_timeline_events",
		// Memory & Heap
		"memory_start_tracking", "memory_stop_tracking",
		"heap_snapshot", "heap_start_tracking", "heap_stop_tracking", "heap_gc",
		// Profiler
		"cpu_start_profiling", "cpu_stop_profiling",
		"script_start_profiling", "script_stop_profiling",
		// Animation
		"animation_enable", "animation_disable",
		"animation_start_tracking", "animation_stop_tracking",
		"get_animation_effect", "resolve_animation",
		// Canvas
		"canvas_enable", "canvas_disable",
		"get_canvas_content", "start_canvas_recording", "stop_canvas_recording",
		"get_shader_source",
		// LayerTree
		"get_layer_tree", "get_compositing_reasons",
		// Workers
		"worker_enable", "worker_disable", "send_to_worker",
		"get_service_worker_info",
		// Audit & Security
		"run_audit", "get_certificate_info",
		// Browser
		"browser_extensions_enable", "browser_extensions_disable",
		// Interaction
		"click", "fill", "type_text",
	}
	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("tool %q not found in tools list", name)
		}
	}
}

func TestMCP_CallToolWithoutSession(t *testing.T) {
	p := startMCP(t)
	p.initialize(t)

	// Try calling navigate without a page selected.
	p.send(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "navigate",
			"arguments": map[string]interface{}{
				"url": "https://example.com",
			},
		},
	})

	resp := p.recv(t)

	// MCP SDK returns tool errors as isError:true in result content.
	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T: %v", resp["result"], resp)
	}

	isError, _ := result["isError"].(bool)
	content, _ := result["content"].([]interface{})

	if len(content) > 0 {
		first, ok := content[0].(map[string]interface{})
		if ok {
			text, _ := first["text"].(string)
			if strings.Contains(text, "select_page") || strings.Contains(text, "no page") {
				t.Logf("got expected error: %s", text)
			} else if !isError {
				t.Errorf("expected error about no page selected, got: %s", text)
			}
		}
	}
}

func TestMCP_ListDevicesTool(t *testing.T) {
	p := startMCP(t)
	p.initialize(t)

	// list_devices does HTTP to iwdp directly — doesn't require a session.
	p.send(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "list_devices",
			"arguments": map[string]interface{}{},
		},
	})

	resp := p.recv(t)

	if resp["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", resp["jsonrpc"])
	}
	if resp["id"] != float64(2) {
		t.Errorf("id = %v, want 2", resp["id"])
	}

	// Response should have result (even if the tool returns an error, MCP wraps it).
	if resp["result"] == nil && resp["error"] == nil {
		t.Error("expected either result or error in response")
	}
}

func TestMCP_ListPagesTool(t *testing.T) {
	p := startMCP(t)
	p.initialize(t)

	// list_pages does HTTP to a device port directly — doesn't require a session.
	p.send(t, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "list_pages",
			"arguments": map[string]interface{}{},
		},
	})

	resp := p.recv(t)

	if resp["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", resp["jsonrpc"])
	}
	// Should get a valid response regardless of whether iwdp is running.
	if resp["result"] == nil && resp["error"] == nil {
		t.Error("expected either result or error")
	}
}
