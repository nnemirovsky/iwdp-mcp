package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nnemirovsky/iwdp-mcp/internal/proxy"
	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// session holds the active WebKit client and collectors.
type session struct {
	mu                         sync.Mutex
	client                     *webkit.Client
	networkMonitor             *tools.NetworkMonitor
	consoleCollector           *tools.ConsoleCollector
	timelineCollector          *tools.TimelineCollector
	interceptionCollector      *tools.InterceptionCollector
	cpuProfilerCollector       *tools.CPUProfilerCollector
	scriptProfilerCollector    *tools.ScriptProfilerCollector
	memoryTrackingCollector    *tools.MemoryTrackingCollector
	heapTrackingCollector      *tools.HeapTrackingCollector
	animationTrackingCollector *tools.AnimationTrackingCollector
}

var sess session

func getClient(ctx context.Context) (*webkit.Client, error) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if sess.client != nil {
		return sess.client, nil
	}
	return nil, fmt.Errorf("no page selected — use select_page first")
}

// lookupInterceptStage finds the stage for an intercepted request from the collector.
func lookupInterceptStage(requestID string) string {
	sess.mu.Lock()
	ic := sess.interceptionCollector
	sess.mu.Unlock()
	if ic == nil {
		return "request"
	}
	for _, req := range ic.GetPending() {
		if req.RequestID == requestID {
			return req.Stage
		}
	}
	return "request"
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "iwdp-mcp",
		Version: "0.4.0",
	}, nil)

	registerTools(server)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

// --- Input/Output types for MCP tools ---

type EmptyInput struct{}

type (
	IWDPStatusInput struct {
		AutoStart *bool `json:"auto_start,omitempty" jsonschema:"if true, automatically start iwdp when not running (default: true)"`
	}

	ListDevicesInput  struct{}
	ListDevicesOutput struct {
		Devices []webkit.DeviceEntry `json:"devices"`
	}
)

type ListPagesInput struct {
	Port int `json:"port,omitempty" jsonschema:"device port (from list_devices URL), defaults to 9222. Each device gets an incremented port starting at 9222."`
}
type ListPagesOutput struct {
	Pages []webkit.PageEntry `json:"pages"`
}

type SelectPageInput struct {
	WebSocketURL string `json:"websocket_url" jsonschema:"WebSocket debugger URL from list_pages"`
}
type SelectPageOutput struct {
	Connected bool   `json:"connected"`
	URL       string `json:"url"`
}

type NavigateInput struct {
	URL string `json:"url" jsonschema:"URL to navigate to"`
}

type (
	TakeScreenshotInput  struct{}
	TakeScreenshotOutput struct {
		FilePath string `json:"file_path"`
	}
)

type SnapshotNodeInput struct {
	NodeID int `json:"node_id" jsonschema:"DOM node ID to snapshot"`
}

type EvaluateScriptInput struct {
	Expression    string `json:"expression" jsonschema:"JavaScript expression to evaluate"`
	ReturnByValue bool   `json:"return_by_value,omitempty" jsonschema:"return result by value instead of reference"`
}
type EvaluateScriptOutput struct {
	Result any    `json:"result"`
	Type   string `json:"type"`
}

type CallFunctionInput struct {
	ObjectID            string        `json:"object_id" jsonschema:"remote object ID"`
	FunctionDeclaration string        `json:"function_declaration" jsonschema:"function source code"`
	Arguments           []interface{} `json:"arguments,omitempty" jsonschema:"function arguments"`
	ReturnByValue       bool          `json:"return_by_value,omitempty"`
}

type GetPropertiesInput struct {
	ObjectID      string `json:"object_id" jsonschema:"remote object ID"`
	OwnProperties bool   `json:"own_properties,omitempty" jsonschema:"only own properties"`
}

type GetDocumentInput struct {
	Depth int `json:"depth,omitempty" jsonschema:"max depth to return, 0 for full tree"`
}

type QuerySelectorInput struct {
	NodeID   int    `json:"node_id" jsonschema:"parent node ID"`
	Selector string `json:"selector" jsonschema:"CSS selector"`
}

type QuerySelectorAllInput struct {
	NodeID   int    `json:"node_id" jsonschema:"parent node ID"`
	Selector string `json:"selector" jsonschema:"CSS selector"`
}
type QuerySelectorAllOutput struct {
	NodeIDs []int `json:"node_ids"`
}

type GetOuterHTMLInput struct {
	NodeID int `json:"node_id" jsonschema:"DOM node ID"`
}
type GetOuterHTMLOutput struct {
	OuterHTML string `json:"outer_html"`
}

type GetAttributesInput struct {
	NodeID int `json:"node_id" jsonschema:"DOM node ID"`
}
type GetAttributesOutput struct {
	Attributes map[string]string `json:"attributes"`
}

type GetEventListenersInput struct {
	NodeID int `json:"node_id" jsonschema:"DOM node ID"`
}

type HighlightNodeInput struct {
	NodeID int `json:"node_id" jsonschema:"DOM node ID to highlight"`
}

type GetMatchedStylesInput struct {
	NodeID int `json:"node_id" jsonschema:"DOM node ID"`
}

type GetComputedStyleInput struct {
	NodeID int `json:"node_id" jsonschema:"DOM node ID"`
}

type GetInlineStylesInput struct {
	NodeID int `json:"node_id" jsonschema:"DOM node ID"`
}

type SetStyleTextInput struct {
	StyleID json.RawMessage `json:"style_id" jsonschema:"CSS style ID from getMatchedStyles"`
	Text    string          `json:"text" jsonschema:"new CSS text"`
}

type GetStylesheetTextInput struct {
	StyleSheetID string `json:"stylesheet_id" jsonschema:"stylesheet ID"`
}

type ForcePseudoStateInput struct {
	NodeID        int      `json:"node_id" jsonschema:"DOM node ID"`
	PseudoClasses []string `json:"pseudo_classes" jsonschema:"pseudo-classes to force: hover, active, focus, visited, focus-within, focus-visible"`
}

type (
	NetworkEnableInput  struct{}
	NetworkDisableInput struct{}
)

type GetResponseBodyInput struct {
	RequestID string `json:"request_id" jsonschema:"network request ID"`
}
type GetResponseBodyOutput struct {
	Body          string `json:"body"`
	Base64Encoded bool   `json:"base64_encoded"`
}

type SetExtraHeadersInput struct {
	Headers map[string]string `json:"headers" jsonschema:"HTTP headers to set"`
}

type SetRequestInterceptionInput struct {
	Enabled    bool   `json:"enabled" jsonschema:"enable or disable request interception"`
	URLPattern string `json:"url_pattern,omitempty" jsonschema:"URL pattern to intercept (empty = all requests)"`
	Stage      string `json:"stage,omitempty" jsonschema:"interception stage: request or response (default: request)"`
	IsRegex    bool   `json:"is_regex,omitempty" jsonschema:"treat url_pattern as regex"`
}

type InterceptContinueInput struct {
	RequestID string `json:"request_id" jsonschema:"intercepted request ID"`
}

type InterceptWithResponseInput struct {
	RequestID     string            `json:"request_id" jsonschema:"intercepted request ID"`
	StatusCode    int               `json:"status_code" jsonschema:"HTTP status code"`
	Headers       map[string]string `json:"headers,omitempty" jsonschema:"response headers"`
	Content       string            `json:"content,omitempty" jsonschema:"response body content"`
	Base64Encoded bool              `json:"base64_encoded,omitempty" jsonschema:"whether content is base64-encoded"`
}

type SetEmulatedConditionsInput struct {
	BytesPerSecondLimit int `json:"bytes_per_second_limit" jsonschema:"bytes per second limit"`
}

type SetResourceCachingDisabledInput struct {
	Disabled bool `json:"disabled" jsonschema:"disable caching"`
}

type (
	GetCookiesInput struct{}
	SetCookieInput  struct {
		Name     string  `json:"name"`
		Value    string  `json:"value"`
		Domain   string  `json:"domain,omitempty"`
		Path     string  `json:"path,omitempty"`
		Expires  float64 `json:"expires,omitempty" jsonschema:"expiry as Unix timestamp in seconds (0 or omit for session cookie)"`
		Secure   bool    `json:"secure,omitempty"`
		HTTPOnly bool    `json:"http_only,omitempty"`
		SameSite string  `json:"same_site,omitempty" jsonschema:"cookie SameSite attribute: None, Lax, or Strict (default: Lax)"`
	}
)

type DeleteCookieInput struct {
	Name string `json:"name" jsonschema:"cookie name"`
	URL  string `json:"url" jsonschema:"cookie URL"`
}

type StorageInput struct {
	SecurityOrigin string `json:"security_origin" jsonschema:"page origin, e.g. https://example.com"`
}
type StorageItemInput struct {
	SecurityOrigin string `json:"security_origin"`
	Key            string `json:"key"`
	Value          string `json:"value,omitempty"`
}
type StorageRemoveInput struct {
	SecurityOrigin string `json:"security_origin"`
	Key            string `json:"key"`
}

type IndexedDBInput struct {
	SecurityOrigin  string `json:"security_origin"`
	DatabaseName    string `json:"database_name,omitempty"`
	ObjectStoreName string `json:"object_store_name,omitempty"`
	SkipCount       int    `json:"skip_count,omitempty"`
	PageSize        int    `json:"page_size,omitempty" jsonschema:"number of records to return, default 10"`
}

type (
	ConsoleGetInput  struct{}
	SetLogLevelInput struct {
		Source string `json:"source" jsonschema:"logging channel source"`
		Level  string `json:"level" jsonschema:"log level"`
	}
)

type DebuggerSetBreakpointInput struct {
	URL          string `json:"url" jsonschema:"script URL"`
	LineNumber   int    `json:"line_number" jsonschema:"line number (0-based)"`
	ColumnNumber *int   `json:"column_number,omitempty"`
	Condition    string `json:"condition,omitempty" jsonschema:"breakpoint condition expression"`
}

type BreakpointIDInput struct {
	BreakpointID string `json:"breakpoint_id"`
}

type GetScriptSourceInput struct {
	ScriptID string `json:"script_id"`
}

type SearchInContentInput struct {
	ScriptID      string `json:"script_id"`
	Query         string `json:"query"`
	CaseSensitive bool   `json:"case_sensitive,omitempty"`
	IsRegex       bool   `json:"is_regex,omitempty"`
}

type EvaluateOnCallFrameInput struct {
	CallFrameID   string `json:"call_frame_id"`
	Expression    string `json:"expression"`
	ReturnByValue bool   `json:"return_by_value,omitempty"`
}

type SetPauseOnExceptionsInput struct {
	State string `json:"state" jsonschema:"none, uncaught, or all"`
}

type DOMBreakpointInput struct {
	NodeID int    `json:"node_id"`
	Type   string `json:"type" jsonschema:"subtree-modified, attribute-modified, or node-removed"`
}

type EventBreakpointInput struct {
	BreakpointType string `json:"breakpoint_type" jsonschema:"required,breakpoint type: animation-frame, interval, listener, or timeout"`
	EventName      string `json:"event_name"`
}

type URLBreakpointInput struct {
	URL     string `json:"url"`
	IsRegex bool   `json:"is_regex,omitempty"`
}

type ReloadInput struct {
	IgnoreCache bool `json:"ignore_cache,omitempty" jsonschema:"if true, reload ignoring cached resources"`
}

type RemoveURLBreakpointInput struct {
	URL string `json:"url"`
}

type TimelineStartInput struct {
	MaxCallStackDepth int `json:"max_call_stack_depth,omitempty"`
}

type HeapSnapshotInput struct{}

type AnimationIDInput struct {
	AnimationID string `json:"animation_id"`
	ObjectGroup string `json:"object_group,omitempty"`
}

type CanvasIDInput struct {
	CanvasID   string `json:"canvas_id"`
	FrameCount int    `json:"frame_count,omitempty" jsonschema:"number of frames to record (0 for unlimited)"`
}

type ShaderSourceInput struct {
	ProgramID  string `json:"program_id"`
	ShaderType string `json:"shader_type" jsonschema:"vertex or fragment"`
}

type LayerNodeInput struct {
	NodeID int `json:"node_id"`
}

type LayerIDInput struct {
	LayerID string `json:"layer_id"`
}

type WorkerMessageInput struct {
	WorkerID string `json:"worker_id"`
	Message  string `json:"message"`
}

type AuditInput struct {
	Test string `json:"test" jsonschema:"audit test specification"`
}

type CertificateInput struct {
	RequestID string `json:"request_id" jsonschema:"network request ID"`
}

type ClickInput struct {
	Selector string `json:"selector" jsonschema:"CSS selector of element to click"`
}

type FillInput struct {
	Selector string `json:"selector" jsonschema:"CSS selector of input element"`
	Value    string `json:"value" jsonschema:"value to fill"`
}

type TypeTextInput struct {
	Text string `json:"text" jsonschema:"text to type into focused element"`
}

// Generic output wrappers
type RawOutput struct {
	Result any `json:"result"`
}

type TextOutput struct {
	Text string `json:"text"`
}

type OKOutput struct {
	OK bool `json:"ok"`
}

type NodeIDOutput struct {
	NodeID int `json:"node_id"`
}

func ok() OKOutput { return OKOutput{OK: true} }

// saveScreenshot saves a data URL (data:image/png;base64,...) to a temp PNG
// file. iOS retina screenshots are too large for inline MCP results — saving
// to disk lets Claude Code read the image directly with its Read tool.
func saveScreenshot(dataURL string) (filePath string, result *mcp.CallToolResult, err error) {
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(dataURL, prefix) {
		return "", &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: dataURL}},
		}, nil
	}
	b64Data := dataURL[len(prefix):]
	rawBytes, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return "", nil, fmt.Errorf("decoding screenshot base64: %w", err)
	}
	tmpDir := filepath.Join(os.TempDir(), "iwdp-mcp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("creating temp dir: %w", err)
	}
	f, err := os.CreateTemp(tmpDir, "screenshot-*.png")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(rawBytes); err != nil {
		return "", nil, fmt.Errorf("writing screenshot: %w", err)
	}
	return f.Name(), &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: fmt.Sprintf("Screenshot saved to %s — use the Read tool to view it.", f.Name()),
		}},
	}, nil
}

// maxInlineResultSize is the maximum number of characters for a tool result
// before it gets saved to a temp file instead. Claude Code truncates results
// above ~60K characters, so we save anything larger to disk.
const maxInlineResultSize = 50_000

// largeResultToFile checks if the JSON representation of result exceeds
// maxInlineResultSize. If so, it saves the JSON to a temp file and returns a
// *mcp.CallToolResult pointing to the file. Otherwise returns nil (use inline).
func largeResultToFile(result any, prefix string) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil || len(data) <= maxInlineResultSize {
		return nil, err
	}
	tmpDir := filepath.Join(os.TempDir(), "iwdp-mcp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	f, err := os.CreateTemp(tmpDir, prefix+"-*.json")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(data); err != nil {
		return nil, fmt.Errorf("writing result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: fmt.Sprintf("Result too large for inline display (%d bytes). Saved to %s — use the Read tool to view it.", len(data), f.Name()),
		}},
	}, nil
}

func registerTools(server *mcp.Server) {
	// --- iwdp status ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "iwdp_status", Description: "Check if ios-webkit-debug-proxy is running and optionally start it. Call this first before any other tool to ensure iwdp is available.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input IWDPStatusInput) (*mcp.CallToolResult, any, error) {
		running := proxy.IsRunning()
		if running {
			return nil, map[string]any{"running": true, "message": "ios-webkit-debug-proxy is running on port 9221"}, nil
		}
		// Default auto_start to true when not explicitly set
		autoStart := input.AutoStart == nil || *input.AutoStart
		if !autoStart {
			return nil, map[string]any{"running": false, "message": "ios-webkit-debug-proxy is not running. Start it with: ios_webkit_debug_proxy --no-frontend"}, nil
		}
		if err := proxy.Start(ctx); err != nil {
			return nil, map[string]any{"running": false, "message": fmt.Sprintf("failed to start iwdp: %v", err)}, nil
		}
		return nil, map[string]any{"running": true, "started": true, "message": "ios-webkit-debug-proxy was not running — started it automatically"}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "restart_iwdp", Description: "Restart ios-webkit-debug-proxy. Use this to recover after a crash (e.g., after a large heap snapshot kills the connection).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		// Also disconnect any active WebSocket client since the old connection is dead
		sess.mu.Lock()
		if sess.client != nil {
			_ = sess.client.Close()
			sess.client = nil
		}
		sess.mu.Unlock()
		if err := proxy.Restart(ctx); err != nil {
			return nil, map[string]any{"restarted": false, "message": fmt.Sprintf("failed to restart iwdp: %v", err)}, nil
		}
		return nil, map[string]any{"restarted": true, "message": "ios-webkit-debug-proxy restarted. Use list_devices and select_page to reconnect."}, nil
	})

	// --- Device/Page management ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_devices", Description: "List connected iOS devices (from iwdp listing port 9221). Each device's URL shows which port to use for list_pages.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ ListDevicesInput) (*mcp.CallToolResult, any, error) {
		if err := proxy.EnsureRunning(ctx); err != nil {
			return nil, ListDevicesOutput{}, err
		}
		devices, err := proxy.ListDevices()
		if err != nil {
			return nil, ListDevicesOutput{}, err
		}
		return nil, ListDevicesOutput{Devices: devices}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "list_pages", Description: "List open Safari tabs/pages on a device port. Use list_devices to find each device's port (default: 9222 for first device, 9223 for second, etc.)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListPagesInput) (*mcp.CallToolResult, any, error) {
		if err := proxy.EnsureRunning(ctx); err != nil {
			return nil, ListPagesOutput{}, err
		}
		port := input.Port
		if port == 0 {
			port = proxy.DefaultFirstDevicePort
		}
		pages, err := proxy.ListPages(port)
		if err != nil {
			return nil, ListPagesOutput{}, err
		}
		return nil, ListPagesOutput{Pages: pages}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "select_page", Description: "Connect to a specific Safari tab by its WebSocket URL",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SelectPageInput) (*mcp.CallToolResult, any, error) {
		// Validate the WebSocket URL points to localhost only.
		parsed, err := url.Parse(input.WebSocketURL)
		if err != nil {
			return nil, SelectPageOutput{}, fmt.Errorf("invalid WebSocket URL: %w", err)
		}
		host := parsed.Hostname()
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			return nil, SelectPageOutput{}, fmt.Errorf("WebSocket URL must point to localhost, 127.0.0.1, or ::1; got %q", host)
		}

		// Create the new client first (outside lock) to avoid holding the
		// lock during a potentially slow network operation.
		newClient, err := webkit.NewClient(ctx, input.WebSocketURL)
		if err != nil {
			return nil, SelectPageOutput{}, err
		}

		// Atomically swap: close old client and set the new one under a single lock.
		sess.mu.Lock()
		oldClient := sess.client
		sess.client = newClient
		sess.networkMonitor = nil
		sess.consoleCollector = nil
		sess.timelineCollector = nil
		sess.interceptionCollector = nil
		sess.mu.Unlock()

		if oldClient != nil {
			_ = oldClient.Close()
		}

		return nil, SelectPageOutput{Connected: true, URL: input.WebSocketURL}, nil
	})

	// --- Page ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "navigate", Description: "Navigate to a URL",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input NavigateInput) (*mcp.CallToolResult, any, error) {
		// Validate URL scheme — only http and https are allowed.
		parsed, err := url.Parse(input.URL)
		if err != nil {
			return nil, OKOutput{}, fmt.Errorf("invalid URL: %w", err)
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "http" && scheme != "https" {
			return nil, OKOutput{}, fmt.Errorf("unsupported URL scheme %q; only http and https are allowed", parsed.Scheme)
		}
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.Navigate(ctx, c, input.URL)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "reload", Description: "Reload the current page",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ReloadInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.Reload(ctx, c, input.IgnoreCache)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "take_screenshot", Description: "Capture page screenshot as PNG file. Returns the file path — use the Read tool to view it.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ TakeScreenshotInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, TakeScreenshotOutput{}, err
		}
		dataURL, err := tools.TakeScreenshot(ctx, c)
		if err != nil {
			return nil, TakeScreenshotOutput{}, err
		}
		path, result, err := saveScreenshot(dataURL)
		if err != nil {
			return nil, TakeScreenshotOutput{}, err
		}
		return result, TakeScreenshotOutput{FilePath: path}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "snapshot_node", Description: "Capture a specific DOM node as PNG file. Returns the file path — use the Read tool to view it.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SnapshotNodeInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, TakeScreenshotOutput{}, err
		}
		dataURL, err := tools.SnapshotNode(ctx, c, input.NodeID)
		if err != nil {
			return nil, TakeScreenshotOutput{}, err
		}
		path, result, err := saveScreenshot(dataURL)
		if err != nil {
			return nil, TakeScreenshotOutput{}, err
		}
		return result, TakeScreenshotOutput{FilePath: path}, nil
	})

	// --- Runtime ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "evaluate_script", Description: "Evaluate JavaScript expression in page context",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input EvaluateScriptInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, EvaluateScriptOutput{}, err
		}
		result, err := tools.EvaluateScript(ctx, c, input.Expression, input.ReturnByValue)
		if err != nil {
			return nil, EvaluateScriptOutput{}, err
		}
		out := EvaluateScriptOutput{Result: result.Result, Type: result.Result.Type}
		if fileResult, err := largeResultToFile(out, "eval"); err == nil && fileResult != nil {
			return fileResult, nil, nil
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "call_function", Description: "Call a function on a remote object",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CallFunctionInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		result, err := tools.CallFunctionOn(ctx, c, input.ObjectID, input.FunctionDeclaration, input.Arguments, input.ReturnByValue)
		if err != nil {
			return nil, RawOutput{}, err
		}
		out := RawOutput{Result: result.Result}
		if fileResult, err := largeResultToFile(out, "call-fn"); err == nil && fileResult != nil {
			return fileResult, nil, nil
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_properties", Description: "Get properties of a remote object",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetPropertiesInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		props, err := tools.GetProperties(ctx, c, input.ObjectID, input.OwnProperties)
		if err != nil {
			return nil, RawOutput{}, err
		}
		out := RawOutput{Result: props}
		if fileResult, err := largeResultToFile(out, "props"); err == nil && fileResult != nil {
			return fileResult, nil, nil
		}
		return nil, out, nil
	})

	// --- DOM ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_document", Description: "Get the DOM tree",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetDocumentInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		doc, err := tools.GetDocument(ctx, c, input.Depth)
		if err != nil {
			return nil, RawOutput{}, err
		}
		out := RawOutput{Result: doc}
		if fileResult, err := largeResultToFile(out, "dom"); err == nil && fileResult != nil {
			return fileResult, nil, nil
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "query_selector", Description: "Find first element matching CSS selector",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input QuerySelectorInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, NodeIDOutput{}, err
		}
		nodeID, err := tools.QuerySelector(ctx, c, input.NodeID, input.Selector)
		if err != nil {
			return nil, NodeIDOutput{}, err
		}
		return nil, NodeIDOutput{NodeID: nodeID}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "query_selector_all", Description: "Find all elements matching CSS selector",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input QuerySelectorAllInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, QuerySelectorAllOutput{}, err
		}
		nodeIDs, err := tools.QuerySelectorAll(ctx, c, input.NodeID, input.Selector)
		if err != nil {
			return nil, QuerySelectorAllOutput{}, err
		}
		return nil, QuerySelectorAllOutput{NodeIDs: nodeIDs}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_outer_html", Description: "Get element outer HTML",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetOuterHTMLInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, GetOuterHTMLOutput{}, err
		}
		html, err := tools.GetOuterHTML(ctx, c, input.NodeID)
		if err != nil {
			return nil, GetOuterHTMLOutput{}, err
		}
		out := GetOuterHTMLOutput{OuterHTML: html}
		if fileResult, err := largeResultToFile(out, "html"); err == nil && fileResult != nil {
			return fileResult, nil, nil
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_attributes", Description: "Get element attributes",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetAttributesInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, GetAttributesOutput{}, err
		}
		attrs, err := tools.GetAttributes(ctx, c, input.NodeID)
		if err != nil {
			return nil, GetAttributesOutput{}, err
		}
		return nil, GetAttributesOutput{Attributes: attrs}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_event_listeners", Description: "Get event listeners on a DOM node",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetEventListenersInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		listeners, err := tools.GetEventListeners(ctx, c, input.NodeID)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: listeners}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "highlight_node", Description: "Highlight a DOM element in the browser",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input HighlightNodeInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.HighlightNode(ctx, c, input.NodeID)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "hide_highlight", Description: "Remove the current DOM highlight overlay",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.HideHighlight(ctx, c)
	})

	// --- CSS ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_matched_styles", Description: "Get matching CSS rules for a node",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetMatchedStylesInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		result, err := tools.GetMatchedStyles(ctx, c, input.NodeID)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: result}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_computed_style", Description: "Get computed style for a node",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetComputedStyleInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		props, err := tools.GetComputedStyle(ctx, c, input.NodeID)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: props}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_inline_styles", Description: "Get inline styles for a node",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetInlineStylesInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		style, err := tools.GetInlineStyles(ctx, c, input.NodeID)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: style}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_style_text", Description: "Modify a CSS style declaration",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SetStyleTextInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		style, err := tools.SetStyleText(ctx, c, input.StyleID, input.Text)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: style}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_all_stylesheets", Description: "List all stylesheets",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		sheets, err := tools.GetAllStylesheets(ctx, c)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: sheets}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_stylesheet_text", Description: "Get stylesheet source text",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetStylesheetTextInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, TextOutput{}, err
		}
		text, err := tools.GetStylesheetText(ctx, c, input.StyleSheetID)
		if err != nil {
			return nil, TextOutput{}, err
		}
		return nil, TextOutput{Text: text}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "force_pseudo_state", Description: "Force pseudo-class state on a node (:hover, :active, :focus, etc.)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ForcePseudoStateInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.ForcePseudoState(ctx, c, input.NodeID, input.PseudoClasses)
	})

	// --- Network ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "network_enable", Description: "Start network monitoring",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ NetworkEnableInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		sess.mu.Lock()
		if sess.networkMonitor == nil {
			sess.networkMonitor = tools.NewNetworkMonitor()
		}
		nm := sess.networkMonitor
		sess.mu.Unlock()
		return nil, ok(), nm.Start(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "network_disable", Description: "Stop network monitoring",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ NetworkDisableInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		sess.mu.Lock()
		nm := sess.networkMonitor
		sess.mu.Unlock()
		if nm == nil {
			return nil, ok(), nil
		}
		return nil, ok(), nm.Stop(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "list_network_requests", Description: "Get collected network requests",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		sess.mu.Lock()
		nm := sess.networkMonitor
		sess.mu.Unlock()
		if nm == nil {
			return nil, RawOutput{Result: []any{}}, nil
		}
		out := RawOutput{Result: nm.GetRequests()}
		if fileResult, err := largeResultToFile(out, "network"); err == nil && fileResult != nil {
			return fileResult, nil, nil
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_response_body", Description: "Get response body for a network request",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetResponseBodyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, GetResponseBodyOutput{}, err
		}
		body, b64, err := tools.GetResponseBody(ctx, c, input.RequestID)
		if err != nil {
			return nil, GetResponseBodyOutput{}, err
		}
		out := GetResponseBodyOutput{Body: body, Base64Encoded: b64}
		if fileResult, err := largeResultToFile(out, "response"); err == nil && fileResult != nil {
			return fileResult, nil, nil
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_extra_headers", Description: "Set custom HTTP headers for all requests",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SetExtraHeadersInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.SetExtraHeaders(ctx, c, input.Headers)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_request_interception", Description: "Enable or disable request interception. When enabled, intercepted requests appear in list_intercepted_requests and must be continued or responded to.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SetRequestInterceptionInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		if input.Enabled {
			sess.mu.Lock()
			if sess.interceptionCollector == nil {
				sess.interceptionCollector = tools.NewInterceptionCollector()
			}
			ic := sess.interceptionCollector
			sess.mu.Unlock()
			return nil, ok(), ic.Start(ctx, c, input.URLPattern, input.Stage, input.IsRegex)
		}
		sess.mu.Lock()
		ic := sess.interceptionCollector
		sess.mu.Unlock()
		if ic != nil {
			return nil, ok(), ic.Stop(ctx, c)
		}
		return nil, ok(), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "list_intercepted_requests", Description: "List pending intercepted requests. Each has a request_id to use with intercept_continue or intercept_with_response.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		sess.mu.Lock()
		ic := sess.interceptionCollector
		sess.mu.Unlock()
		if ic == nil {
			return nil, RawOutput{Result: []any{}}, nil
		}
		return nil, RawOutput{Result: ic.GetPending()}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "intercept_continue", Description: "Continue an intercepted request without modification",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input InterceptContinueInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		stage := lookupInterceptStage(input.RequestID)
		err = tools.InterceptContinue(ctx, c, input.RequestID, stage)
		if err == nil {
			sess.mu.Lock()
			if sess.interceptionCollector != nil {
				sess.interceptionCollector.RemovePending(input.RequestID)
			}
			sess.mu.Unlock()
		}
		return nil, ok(), err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "intercept_with_response", Description: "Respond to an intercepted request with a custom response",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input InterceptWithResponseInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		stage := lookupInterceptStage(input.RequestID)
		err = tools.InterceptWithResponse(ctx, c, input.RequestID, stage, input.StatusCode, input.Headers, input.Content, input.Base64Encoded)
		if err == nil {
			sess.mu.Lock()
			if sess.interceptionCollector != nil {
				sess.interceptionCollector.RemovePending(input.RequestID)
			}
			sess.mu.Unlock()
		}
		return nil, ok(), err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "intercept_continue_all", Description: "Continue all pending intercepted requests without modification",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		sess.mu.Lock()
		ic := sess.interceptionCollector
		sess.mu.Unlock()
		if ic == nil {
			return nil, RawOutput{Result: map[string]int{"continued": 0}}, nil
		}
		pending := ic.GetPending()
		continued := 0
		for _, r := range pending {
			if err := tools.InterceptContinue(ctx, c, r.RequestID, r.Stage); err == nil {
				ic.RemovePending(r.RequestID)
				continued++
			}
		}
		return nil, RawOutput{Result: map[string]int{"continued": continued}}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "intercept_block_all", Description: "Block all pending intercepted requests with 403 Forbidden",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		sess.mu.Lock()
		ic := sess.interceptionCollector
		sess.mu.Unlock()
		if ic == nil {
			return nil, RawOutput{Result: map[string]int{"blocked": 0}}, nil
		}
		pending := ic.GetPending()
		blocked := 0
		for _, r := range pending {
			if err := tools.InterceptWithResponse(ctx, c, r.RequestID, r.Stage, 403, map[string]string{"Content-Type": "text/plain"}, "Blocked", false); err == nil {
				ic.RemovePending(r.RequestID)
				blocked++
			}
		}
		return nil, RawOutput{Result: map[string]int{"blocked": blocked}}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_emulated_conditions", Description: "Throttle network speed",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SetEmulatedConditionsInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.SetEmulatedConditions(ctx, c, input.BytesPerSecondLimit)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_resource_caching_disabled", Description: "Disable resource caching",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SetResourceCachingDisabledInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.SetResourceCachingDisabled(ctx, c, input.Disabled)
	})

	// --- Cookies ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_cookies", Description: "Get all cookies including httpOnly and secure",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ GetCookiesInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		cookies, err := tools.GetCookies(ctx, c)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: cookies}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_cookie", Description: "Set a cookie",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SetCookieInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		sameSite := input.SameSite
		if sameSite == "" {
			sameSite = "Lax"
		}
		cookie := webkit.Cookie{
			Name:     input.Name,
			Value:    input.Value,
			Domain:   input.Domain,
			Path:     input.Path,
			Expires:  input.Expires,
			Session:  input.Expires == 0,
			Secure:   input.Secure,
			HTTPOnly: input.HTTPOnly,
			SameSite: sameSite,
		}
		return nil, ok(), tools.SetCookie(ctx, c, cookie)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "delete_cookie", Description: "Delete a cookie",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteCookieInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.DeleteCookie(ctx, c, input.Name, input.URL)
	})

	// --- DOM Storage ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_local_storage", Description: "Get localStorage items",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StorageInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		items, err := tools.GetLocalStorage(ctx, c, input.SecurityOrigin)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: items}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_local_storage_item", Description: "Set a localStorage item",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StorageItemInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.SetLocalStorageItem(ctx, c, input.SecurityOrigin, input.Key, input.Value)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "remove_local_storage_item", Description: "Remove a localStorage item",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StorageRemoveInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.RemoveLocalStorageItem(ctx, c, input.SecurityOrigin, input.Key)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "clear_local_storage", Description: "Clear all localStorage",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StorageInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.ClearLocalStorage(ctx, c, input.SecurityOrigin)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_session_storage", Description: "Get sessionStorage items",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StorageInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		items, err := tools.GetSessionStorage(ctx, c, input.SecurityOrigin)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: items}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_session_storage_item", Description: "Set a sessionStorage item",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StorageItemInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.SetSessionStorageItem(ctx, c, input.SecurityOrigin, input.Key, input.Value)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "remove_session_storage_item", Description: "Remove a sessionStorage item",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StorageRemoveInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.RemoveSessionStorageItem(ctx, c, input.SecurityOrigin, input.Key)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "clear_session_storage", Description: "Clear all sessionStorage",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StorageInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.ClearSessionStorage(ctx, c, input.SecurityOrigin)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "list_indexed_databases", Description: "List IndexedDB databases",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StorageInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		dbs, err := tools.ListIndexedDatabases(ctx, c, input.SecurityOrigin)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: dbs}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_indexed_db_data", Description: "Query IndexedDB object store data",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input IndexedDBInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		pageSize := input.PageSize
		if pageSize == 0 {
			pageSize = 10
		}
		result, err := tools.GetIndexedDBData(ctx, c, input.SecurityOrigin, input.DatabaseName, input.ObjectStoreName, input.SkipCount, pageSize)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: result}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "clear_indexed_db_store", Description: "Clear an IndexedDB object store",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input IndexedDBInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.ClearIndexedDBStore(ctx, c, input.SecurityOrigin, input.DatabaseName, input.ObjectStoreName)
	})

	// --- Console ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "console_enable", Description: "Start collecting console messages",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		sess.mu.Lock()
		if sess.consoleCollector == nil {
			sess.consoleCollector = tools.NewConsoleCollector()
		}
		cc := sess.consoleCollector
		sess.mu.Unlock()
		return nil, ok(), cc.Start(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "console_disable", Description: "Stop collecting console messages",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		sess.mu.Lock()
		cc := sess.consoleCollector
		sess.mu.Unlock()
		if cc == nil {
			return nil, ok(), nil
		}
		return nil, ok(), cc.Stop(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_console_messages", Description: "Get collected console messages",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ ConsoleGetInput) (*mcp.CallToolResult, any, error) {
		sess.mu.Lock()
		cc := sess.consoleCollector
		sess.mu.Unlock()
		if cc == nil {
			return nil, RawOutput{Result: []any{}}, nil
		}
		out := RawOutput{Result: cc.GetMessages()}
		if fileResult, err := largeResultToFile(out, "console"); err == nil && fileResult != nil {
			return fileResult, nil, nil
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "clear_console", Description: "Clear console messages",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		sess.mu.Lock()
		if sess.consoleCollector != nil {
			sess.consoleCollector.Clear()
		}
		sess.mu.Unlock()
		return nil, ok(), tools.ClearConsoleMessages(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_log_level", Description: "Set logging channel level",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SetLogLevelInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.SetLogLevel(ctx, c, input.Source, input.Level)
	})

	// --- Debugger ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "debugger_enable", Description: "Enable JavaScript debugger",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.DebuggerEnable(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "debugger_disable", Description: "Disable JavaScript debugger",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.DebuggerDisable(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_breakpoint", Description: "Set a breakpoint by URL and line number",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DebuggerSetBreakpointInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		bpID, locs, err := tools.SetBreakpointByURL(ctx, c, input.URL, input.LineNumber, input.ColumnNumber, input.Condition)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: map[string]interface{}{"breakpointId": bpID, "locations": locs}}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "remove_breakpoint", Description: "Remove a breakpoint",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input BreakpointIDInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.RemoveBreakpoint(ctx, c, webkit.BreakpointID(input.BreakpointID))
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "pause", Description: "Pause JavaScript execution",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.Pause(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "resume", Description: "Resume JavaScript execution",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.Resume(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "step_over", Description: "Step over current statement",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.StepOver(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "step_into", Description: "Step into function call",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.StepInto(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "step_out", Description: "Step out of current function",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.StepOut(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_script_source", Description: "Get source code of a loaded script",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetScriptSourceInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, TextOutput{}, err
		}
		src, err := tools.GetScriptSource(ctx, c, input.ScriptID)
		if err != nil {
			return nil, TextOutput{}, err
		}
		return nil, TextOutput{Text: src}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "search_in_content", Description: "Search within script content",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SearchInContentInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		result, err := tools.SearchInContent(ctx, c, input.ScriptID, input.Query, input.CaseSensitive, input.IsRegex)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: result}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "evaluate_on_call_frame", Description: "Evaluate expression in paused call frame context",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input EvaluateOnCallFrameInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		result, err := tools.EvaluateOnCallFrame(ctx, c, input.CallFrameID, input.Expression, input.ReturnByValue)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: result}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_pause_on_exceptions", Description: "Configure when to pause on exceptions (none, uncaught, all)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SetPauseOnExceptionsInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.SetPauseOnExceptions(ctx, c, input.State)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_dom_breakpoint", Description: "Break on DOM modification (subtree-modified, attribute-modified, node-removed)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DOMBreakpointInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.SetDOMBreakpoint(ctx, c, input.NodeID, input.Type)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "remove_dom_breakpoint", Description: "Remove a DOM modification breakpoint",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DOMBreakpointInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.RemoveDOMBreakpoint(ctx, c, input.NodeID, input.Type)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_event_breakpoint", Description: "Break on event listener",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input EventBreakpointInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.SetEventBreakpoint(ctx, c, input.BreakpointType, input.EventName)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "remove_event_breakpoint", Description: "Remove an event listener breakpoint",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input EventBreakpointInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.RemoveEventBreakpoint(ctx, c, input.BreakpointType, input.EventName)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_url_breakpoint", Description: "Break on URL request",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input URLBreakpointInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.SetURLBreakpoint(ctx, c, input.URL, input.IsRegex)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "remove_url_breakpoint", Description: "Remove a URL request breakpoint",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input RemoveURLBreakpointInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.RemoveURLBreakpoint(ctx, c, input.URL)
	})

	// --- Timeline ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "timeline_start", Description: "Start timeline recording",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input TimelineStartInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		sess.mu.Lock()
		if sess.timelineCollector == nil {
			sess.timelineCollector = tools.NewTimelineCollector()
		}
		tc := sess.timelineCollector
		sess.mu.Unlock()
		return nil, ok(), tc.Start(ctx, c, input.MaxCallStackDepth)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "timeline_stop", Description: "Stop timeline recording",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		sess.mu.Lock()
		tc := sess.timelineCollector
		sess.mu.Unlock()
		if tc == nil {
			return nil, ok(), nil
		}
		return nil, ok(), tc.Stop(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_timeline_events", Description: "Get recorded timeline events",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		sess.mu.Lock()
		tc := sess.timelineCollector
		sess.mu.Unlock()
		if tc == nil {
			return nil, RawOutput{Result: []any{}}, nil
		}
		out := RawOutput{Result: tc.GetEvents()}
		if fileResult, err := largeResultToFile(out, "timeline"); err == nil && fileResult != nil {
			return fileResult, nil, nil
		}
		return nil, out, nil
	})

	// --- Memory & Heap ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "memory_start_tracking", Description: "Start tracking memory usage — collects memory category events",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		sess.mu.Lock()
		if sess.memoryTrackingCollector == nil {
			sess.memoryTrackingCollector = tools.NewMemoryTrackingCollector()
		}
		collector := sess.memoryTrackingCollector
		sess.mu.Unlock()
		return nil, ok(), collector.Start(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "memory_stop_tracking", Description: "Stop tracking memory usage and get collected memory events",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		sess.mu.Lock()
		collector := sess.memoryTrackingCollector
		sess.mu.Unlock()
		if collector == nil {
			return nil, RawOutput{}, fmt.Errorf("memory tracking not started — use memory_start_tracking first")
		}
		result, err := collector.Stop(ctx, c)
		if err != nil {
			return nil, RawOutput{}, err
		}
		if fileResult, ferr := largeResultToFile(result, "memory-tracking"); fileResult != nil {
			return fileResult, nil, ferr
		}
		data, _ := json.Marshal(result)
		return nil, RawOutput{Result: data}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "heap_snapshot", Description: "Take a heap snapshot and save to file. Warning: can be very large (50-200+ MB) on heavy pages and may take minutes.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ HeapSnapshotInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, TextOutput{}, err
		}
		filePath, err := tools.HeapSnapshot(ctx, c)
		if err != nil {
			return nil, TextOutput{}, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("Heap snapshot saved to %s — use the Read tool to view it.", filePath),
			}},
		}, TextOutput{Text: filePath}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "heap_start_tracking", Description: "Start tracking heap allocations — collects GC events. Waits up to 5s to confirm event pipeline health.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		sess.mu.Lock()
		if sess.heapTrackingCollector == nil {
			sess.heapTrackingCollector = tools.NewHeapTrackingCollector()
		}
		collector := sess.heapTrackingCollector
		sess.mu.Unlock()
		if err := collector.Start(ctx, c); err != nil {
			return nil, OKOutput{}, err
		}
		if collector.PipelineHealthy() {
			return nil, struct {
				OK      bool   `json:"ok"`
				Message string `json:"message"`
			}{true, "Heap tracking started. Event pipeline confirmed healthy — GC events will be captured."}, nil
		}
		return nil, struct {
			OK      bool   `json:"ok"`
			Warning string `json:"warning"`
		}{true, "Heap tracking started, but trackingStart event not received (iwdp may not support the 50-200MB snapshot relay). GC events may not be captured. Use heap_snapshot and heap_gc directly instead."}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "heap_stop_tracking", Description: "Stop tracking heap allocations and get collected heap events",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		sess.mu.Lock()
		collector := sess.heapTrackingCollector
		sess.mu.Unlock()
		if collector == nil {
			return nil, RawOutput{}, fmt.Errorf("heap tracking not started — use heap_start_tracking first")
		}
		result, err := collector.Stop(ctx, c)
		if err != nil {
			return nil, RawOutput{}, err
		}
		if fileResult, ferr := largeResultToFile(result, "heap-tracking"); fileResult != nil {
			return fileResult, nil, ferr
		}
		data, _ := json.Marshal(result)
		return nil, RawOutput{Result: data}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "heap_gc", Description: "Force garbage collection",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.HeapGC(ctx, c)
	})

	// --- Profiler ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "cpu_start_profiling", Description: "Start CPU profiling — collects CPU usage samples via events",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		sess.mu.Lock()
		if sess.cpuProfilerCollector == nil {
			sess.cpuProfilerCollector = tools.NewCPUProfilerCollector()
		}
		collector := sess.cpuProfilerCollector
		sess.mu.Unlock()
		return nil, ok(), collector.Start(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "cpu_stop_profiling", Description: "Stop CPU profiling and get collected CPU usage events",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		sess.mu.Lock()
		collector := sess.cpuProfilerCollector
		sess.mu.Unlock()
		if collector == nil {
			return nil, RawOutput{}, fmt.Errorf("CPU profiling not started — use cpu_start_profiling first")
		}
		result, err := collector.Stop(ctx, c)
		if err != nil {
			return nil, RawOutput{}, err
		}
		if fileResult, ferr := largeResultToFile(result, "cpu-profile"); fileResult != nil {
			return fileResult, nil, ferr
		}
		data, _ := json.Marshal(result)
		return nil, RawOutput{Result: data}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "script_start_profiling", Description: "Start script execution profiling with stack sampling",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		sess.mu.Lock()
		if sess.scriptProfilerCollector == nil {
			sess.scriptProfilerCollector = tools.NewScriptProfilerCollector()
		}
		collector := sess.scriptProfilerCollector
		sess.mu.Unlock()
		return nil, ok(), collector.Start(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "script_stop_profiling", Description: "Stop script profiling and get execution events + stack samples",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		sess.mu.Lock()
		collector := sess.scriptProfilerCollector
		sess.mu.Unlock()
		if collector == nil {
			return nil, RawOutput{}, fmt.Errorf("script profiling not started — use script_start_profiling first")
		}
		result, err := collector.Stop(ctx, c)
		if err != nil {
			return nil, RawOutput{}, err
		}
		if fileResult, ferr := largeResultToFile(result, "script-profile"); fileResult != nil {
			return fileResult, nil, ferr
		}
		data, _ := json.Marshal(result)
		return nil, RawOutput{Result: data}, nil
	})

	// --- Animation ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "animation_enable", Description: "Enable animation tracking",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.AnimationEnable(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "animation_disable", Description: "Disable animation tracking",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.AnimationDisable(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "animation_start_tracking", Description: "Start animation profiling — collects animation events",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		sess.mu.Lock()
		if sess.animationTrackingCollector == nil {
			sess.animationTrackingCollector = tools.NewAnimationTrackingCollector()
		}
		collector := sess.animationTrackingCollector
		sess.mu.Unlock()
		return nil, ok(), collector.Start(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "animation_stop_tracking", Description: "Stop animation profiling and get collected animation events",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		sess.mu.Lock()
		collector := sess.animationTrackingCollector
		sess.mu.Unlock()
		if collector == nil {
			return nil, RawOutput{}, fmt.Errorf("animation tracking not started — use animation_start_tracking first")
		}
		result, err := collector.Stop(ctx, c)
		if err != nil {
			return nil, RawOutput{}, err
		}
		if fileResult, ferr := largeResultToFile(result, "animation-tracking"); fileResult != nil {
			return fileResult, nil, ferr
		}
		data, _ := json.Marshal(result)
		return nil, RawOutput{Result: data}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_animation_effect", Description: "Get animation effect details",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input AnimationIDInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		result, err := tools.GetAnimationEffect(ctx, c, input.AnimationID)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: result}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "resolve_animation", Description: "Get animation as Runtime remote object",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input AnimationIDInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		obj, err := tools.ResolveAnimation(ctx, c, input.AnimationID, input.ObjectGroup)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: obj}, nil
	})

	// --- Canvas ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "canvas_enable", Description: "Enable canvas tracking",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.CanvasEnable(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "canvas_disable", Description: "Disable canvas tracking",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.CanvasDisable(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_canvas_content", Description: "Get canvas image content",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CanvasIDInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, TextOutput{}, err
		}
		content, err := tools.GetCanvasContent(ctx, c, input.CanvasID)
		if err != nil {
			return nil, TextOutput{}, err
		}
		return nil, TextOutput{Text: content}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "start_canvas_recording", Description: "Record canvas operations",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CanvasIDInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.StartCanvasRecording(ctx, c, input.CanvasID, input.FrameCount)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "stop_canvas_recording", Description: "Stop canvas recording",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CanvasIDInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.StopCanvasRecording(ctx, c, input.CanvasID)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_shader_source", Description: "Get WebGL shader source code",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ShaderSourceInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, TextOutput{}, err
		}
		src, err := tools.GetShaderSource(ctx, c, input.ProgramID, input.ShaderType)
		if err != nil {
			return nil, TextOutput{}, err
		}
		return nil, TextOutput{Text: src}, nil
	})

	// --- LayerTree ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_layer_tree", Description: "Get compositing layers for a node",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input LayerNodeInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		result, err := tools.GetLayerTree(ctx, c, input.NodeID)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: result}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_compositing_reasons", Description: "Get reasons why a layer was composited",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input LayerIDInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		result, err := tools.GetCompositingReasons(ctx, c, input.LayerID)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: result}, nil
	})

	// --- Workers ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "worker_enable", Description: "Enable web worker tracking",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.WorkerEnable(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "worker_disable", Description: "Disable web worker tracking",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.WorkerDisable(ctx, c)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "send_to_worker", Description: "Send message to a web worker",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input WorkerMessageInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.SendToWorker(ctx, c, input.WorkerID, input.Message)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_service_worker_info", Description: "Get service worker initialization info",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		result, err := tools.GetServiceWorkerInfo(ctx, c)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: result}, nil
	})

	// --- Audit ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "run_audit", Description: "Run a WebKit audit",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input AuditInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		result, err := tools.RunAudit(ctx, c, input.Test)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: result}, nil
	})

	// --- Security ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_certificate_info", Description: "Get TLS certificate info for a network request",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CertificateInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, RawOutput{}, err
		}
		result, err := tools.GetCertificateInfo(ctx, c, input.RequestID)
		if err != nil {
			return nil, RawOutput{}, err
		}
		return nil, RawOutput{Result: result}, nil
	})

	// --- Browser ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "browser_extensions_enable", Description: "Enable browser extensions",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), c.Enable(ctx, "Browser")
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "browser_extensions_disable", Description: "Disable browser extensions",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), c.Disable(ctx, "Browser")
	})

	// --- Element Interaction ---
	mcp.AddTool(server, &mcp.Tool{
		Name: "click", Description: "Click an element by CSS selector",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ClickInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.Click(ctx, c, input.Selector)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "fill", Description: "Fill an input field by CSS selector",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input FillInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.Fill(ctx, c, input.Selector, input.Value)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "type_text", Description: "Type text into the currently focused element",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input TypeTextInput) (*mcp.CallToolResult, any, error) {
		c, err := getClient(ctx)
		if err != nil {
			return nil, OKOutput{}, err
		}
		return nil, ok(), tools.TypeText(ctx, c, input.Text)
	})
}
