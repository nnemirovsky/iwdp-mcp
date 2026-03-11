package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/nnemirovsky/iwdp-mcp/internal/proxy"
	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	ctx := context.Background()
	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	// Core
	case "devices":
		cmdDevices(ctx)
	case "pages":
		cmdPages(ctx, args)
	case "status":
		cmdStatus(ctx)
	case "restart-iwdp":
		cmdRestartIWDP(ctx)
	case "eval":
		cmdEval(ctx, args)
	case "navigate":
		cmdNavigate(ctx, args)
	case "reload":
		cmdReload(ctx, args)
	case "screenshot":
		cmdScreenshot(ctx, args)
	case "snapshot-node":
		cmdSnapshotNode(ctx, args)
	case "cookies":
		cmdCookies(ctx, args)
	case "dom":
		cmdDOM(ctx, args)
	case "console":
		cmdConsole(ctx, args)
	case "network":
		cmdNetwork(ctx, args)

	// Runtime
	case "call-function":
		cmdCallFunction(ctx, args)
	case "get-properties":
		cmdGetProperties(ctx, args)

	// DOM
	case "query-selector":
		cmdQuerySelector(ctx, args)
	case "query-selector-all":
		cmdQuerySelectorAll(ctx, args)
	case "get-outer-html":
		cmdGetOuterHTML(ctx, args)
	case "get-attributes":
		cmdGetAttributes(ctx, args)
	case "get-event-listeners":
		cmdGetEventListeners(ctx, args)
	case "highlight-node":
		cmdHighlightNode(ctx, args)
	case "hide-highlight":
		cmdHideHighlight(ctx, args)

	// CSS
	case "get-matched-styles":
		cmdGetMatchedStyles(ctx, args)
	case "get-computed-style":
		cmdGetComputedStyle(ctx, args)
	case "get-inline-styles":
		cmdGetInlineStyles(ctx, args)
	case "set-style-text":
		cmdSetStyleText(ctx, args)
	case "get-all-stylesheets":
		cmdGetAllStylesheets(ctx, args)
	case "get-stylesheet-text":
		cmdGetStylesheetText(ctx, args)
	case "force-pseudo-state":
		cmdForcePseudoState(ctx, args)

	// Interaction
	case "click":
		cmdClick(ctx, args)
	case "fill":
		cmdFill(ctx, args)
	case "type-text":
		cmdTypeText(ctx, args)

	// Storage
	case "set-cookie":
		cmdSetCookie(ctx, args)
	case "delete-cookie":
		cmdDeleteCookie(ctx, args)
	case "get-local-storage":
		cmdGetLocalStorage(ctx, args)
	case "set-local-storage-item":
		cmdSetLocalStorageItem(ctx, args)
	case "remove-local-storage-item":
		cmdRemoveLocalStorageItem(ctx, args)
	case "clear-local-storage":
		cmdClearLocalStorage(ctx, args)
	case "get-session-storage":
		cmdGetSessionStorage(ctx, args)
	case "set-session-storage-item":
		cmdSetSessionStorageItem(ctx, args)
	case "remove-session-storage-item":
		cmdRemoveSessionStorageItem(ctx, args)
	case "clear-session-storage":
		cmdClearSessionStorage(ctx, args)
	case "list-indexed-databases":
		cmdListIndexedDatabases(ctx, args)
	case "get-indexed-db-data":
		cmdGetIndexedDBData(ctx, args)
	case "clear-indexed-db-store":
		cmdClearIndexedDBStore(ctx, args)

	// Network (extended)
	case "get-response-body":
		cmdGetResponseBody(ctx, args)
	case "set-extra-headers":
		cmdSetExtraHeaders(ctx, args)
	case "set-request-interception":
		cmdSetRequestInterception(ctx, args)
	case "intercept-continue":
		cmdInterceptContinue(ctx, args)
	case "intercept-respond":
		cmdInterceptWithResponse(ctx, args)
	case "set-network-conditions":
		cmdSetEmulatedConditions(ctx, args)
	case "disable-cache":
		cmdSetResourceCachingDisabled(ctx, args)

	// Console (extended)
	case "console-messages":
		cmdConsoleMessages(ctx, args)
	case "clear-console":
		cmdClearConsole(ctx, args)
	case "set-log-level":
		cmdSetLogLevel(ctx, args)

	// Debugger
	case "debugger-enable":
		cmdDebuggerEnable(ctx, args)
	case "debugger-disable":
		cmdDebuggerDisable(ctx, args)
	case "set-breakpoint":
		cmdSetBreakpoint(ctx, args)
	case "remove-breakpoint":
		cmdRemoveBreakpoint(ctx, args)
	case "pause":
		cmdPause(ctx, args)
	case "resume":
		cmdResume(ctx, args)
	case "step-over":
		cmdStepOver(ctx, args)
	case "step-into":
		cmdStepInto(ctx, args)
	case "step-out":
		cmdStepOut(ctx, args)
	case "get-script-source":
		cmdGetScriptSource(ctx, args)
	case "search-in-content":
		cmdSearchInContent(ctx, args)
	case "eval-on-frame":
		cmdEvalOnFrame(ctx, args)
	case "set-pause-on-exceptions":
		cmdSetPauseOnExceptions(ctx, args)

	// DOMDebugger
	case "set-dom-breakpoint":
		cmdSetDOMBreakpoint(ctx, args)
	case "remove-dom-breakpoint":
		cmdRemoveDOMBreakpoint(ctx, args)
	case "set-event-breakpoint":
		cmdSetEventBreakpoint(ctx, args)
	case "remove-event-breakpoint":
		cmdRemoveEventBreakpoint(ctx, args)
	case "set-url-breakpoint":
		cmdSetURLBreakpoint(ctx, args)
	case "remove-url-breakpoint":
		cmdRemoveURLBreakpoint(ctx, args)

	// Performance & Profiling
	case "timeline-record":
		cmdTimelineRecord(ctx, args)
	case "memory-track":
		cmdMemoryTrack(ctx, args)
	case "heap-snapshot":
		cmdHeapSnapshot(ctx, args)
	case "heap-track":
		cmdHeapTrack(ctx, args)
	case "heap-gc":
		cmdHeapGC(ctx, args)
	case "cpu-profile":
		cmdCPUProfile(ctx, args)
	case "script-profile":
		cmdScriptProfile(ctx, args)

	// Animation
	case "animation-enable":
		cmdAnimationEnable(ctx, args)
	case "animation-disable":
		cmdAnimationDisable(ctx, args)
	case "animation-track":
		cmdAnimationTrack(ctx, args)
	case "get-animation-effect":
		cmdGetAnimationEffect(ctx, args)
	case "resolve-animation":
		cmdResolveAnimation(ctx, args)

	// Canvas
	case "canvas-enable":
		cmdCanvasEnable(ctx, args)
	case "canvas-disable":
		cmdCanvasDisable(ctx, args)
	case "get-canvas-content":
		cmdGetCanvasContent(ctx, args)
	case "start-canvas-recording":
		cmdStartCanvasRecording(ctx, args)
	case "stop-canvas-recording":
		cmdStopCanvasRecording(ctx, args)
	case "get-shader-source":
		cmdGetShaderSource(ctx, args)

	// LayerTree
	case "get-layer-tree":
		cmdGetLayerTree(ctx, args)
	case "get-compositing-reasons":
		cmdGetCompositingReasons(ctx, args)

	// Workers
	case "worker-enable":
		cmdWorkerEnable(ctx, args)
	case "worker-disable":
		cmdWorkerDisable(ctx, args)
	case "send-to-worker":
		cmdSendToWorker(ctx, args)
	case "get-service-worker-info":
		cmdGetServiceWorkerInfo(ctx, args)

	// Audit & Security
	case "run-audit":
		cmdRunAudit(ctx, args)
	case "get-certificate-info":
		cmdGetCertificateInfo(ctx, args)

	// Browser
	case "browser-extensions-enable":
		cmdBrowserExtensionsEnable(ctx, args)
	case "browser-extensions-disable":
		cmdBrowserExtensionsDisable(ctx, args)

	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `iwdp-cli — iOS Safari debugging CLI

Usage: iwdp-cli <command> [options]

Core:
  devices                          List connected iOS devices
  pages [port]                     List open Safari tabs (default port: 9222)
  status                           Check if ios_webkit_debug_proxy is running
  restart-iwdp                     Restart ios_webkit_debug_proxy
  eval <expr> [ws-url]             Evaluate JavaScript
  navigate <url> [ws-url]          Navigate to URL
  reload [--ignore-cache] [ws-url] Reload page
  screenshot [-o file] [ws-url]    Take screenshot
  snapshot-node <nodeId> [-o file]  Snapshot a DOM node
  cookies [ws-url]                 Show all cookies (incl. httpOnly)
  dom [selector] [ws-url]          Inspect DOM tree or element
  console [ws-url]                 Stream console messages (Ctrl+C to stop)
  network [ws-url]                 Stream network requests (Ctrl+C to stop)

Runtime:
  call-function <objectId> <func> [ws-url]   Call function on object
  get-properties <objectId> [ws-url]         Get object properties

DOM:
  query-selector <selector> [ws-url]         Find first matching element
  query-selector-all <selector> [ws-url]     Find all matching elements
  get-outer-html <nodeId> [ws-url]           Get element outer HTML
  get-attributes <nodeId> [ws-url]           Get element attributes
  get-event-listeners <nodeId> [ws-url]      Get event listeners
  highlight-node <nodeId> [ws-url]           Highlight element in browser
  hide-highlight [ws-url]                    Hide highlight overlay

CSS:
  get-matched-styles <nodeId> [ws-url]       Get matching CSS rules
  get-computed-style <nodeId> [ws-url]       Get computed style
  get-inline-styles <nodeId> [ws-url]        Get inline styles
  set-style-text <styleId> <text> [ws-url]   Modify style declaration
  get-all-stylesheets [ws-url]               List all stylesheets
  get-stylesheet-text <id> [ws-url]          Get stylesheet source
  force-pseudo-state <nodeId> <states>       Force :hover,:focus etc.

Interaction:
  click <selector> [ws-url]                  Click element
  fill <selector> <value> [ws-url]           Fill input field
  type-text <text> [ws-url]                  Type text into focused element

Storage:
  set-cookie <name> <val> <domain> [opts]    Set a cookie
  delete-cookie <name> <url> [ws-url]        Delete a cookie
  get-local-storage <origin> [ws-url]        Get localStorage items
  set-local-storage-item <origin> <k> <v>    Set localStorage item
  remove-local-storage-item <origin> <key>   Remove localStorage item
  clear-local-storage <origin> [ws-url]      Clear all localStorage
  get-session-storage <origin> [ws-url]      Get sessionStorage items
  set-session-storage-item <origin> <k> <v>  Set sessionStorage item
  remove-session-storage-item <origin> <key> Remove sessionStorage item
  clear-session-storage <origin> [ws-url]    Clear all sessionStorage
  list-indexed-databases <origin> [ws-url]   List IndexedDB databases
  get-indexed-db-data <origin> <db> <store>  Query IndexedDB store
  clear-indexed-db-store <origin> <db> <store> Clear IndexedDB store

Network:
  get-response-body <requestId> [ws-url]     Get response body
  set-extra-headers <json> [ws-url]          Set custom HTTP headers
  set-request-interception <true/false>      Enable/disable interception
  intercept-continue <requestId> [opts]      Continue intercepted request
  intercept-respond <id> <status> <body>     Respond to intercepted request
  set-network-conditions <bytes/sec>         Throttle network
  disable-cache <true/false> [ws-url]        Disable resource caching

Console:
  console-messages [-d secs] [ws-url]        Collect console messages
  clear-console [ws-url]                     Clear console
  set-log-level <source> <level> [ws-url]    Set logging level

Debugger:
  debugger-enable [ws-url]                   Enable debugger
  debugger-disable [ws-url]                  Disable debugger
  set-breakpoint <url> <line> [col] [opts]   Set breakpoint by URL
  remove-breakpoint <id> [ws-url]            Remove breakpoint
  pause [ws-url]                             Pause execution
  resume [ws-url]                            Resume execution
  step-over [ws-url]                         Step over
  step-into [ws-url]                         Step into
  step-out [ws-url]                          Step out
  get-script-source <scriptId> [ws-url]      Get script source
  search-in-content <scriptId> <query>       Search in script
  eval-on-frame <frameId> <expr> [ws-url]    Evaluate on call frame
  set-pause-on-exceptions <none|uncaught|all> Configure exception pausing

DOMDebugger:
  set-dom-breakpoint <nodeId> <type>         Break on DOM change
  remove-dom-breakpoint <nodeId> <type>      Remove DOM breakpoint
  set-event-breakpoint <event> [--type t]    Break on event
  remove-event-breakpoint <event> [--type t] Remove event breakpoint
  set-url-breakpoint <url> [--regex]         Break on URL request
  remove-url-breakpoint <url> [ws-url]       Remove URL breakpoint

Performance:
  timeline-record [-d secs] [ws-url]         Record timeline events
  memory-track [-d secs] [ws-url]            Track memory usage
  heap-snapshot [ws-url]                     Take heap snapshot
  heap-track [-d secs] [ws-url]              Track heap allocations
  heap-gc [ws-url]                           Force garbage collection
  cpu-profile [-d secs] [ws-url]             CPU profiling
  script-profile [-d secs] [ws-url]          Script execution profiling

Animation:
  animation-enable [ws-url]                  Enable animation tracking
  animation-disable [ws-url]                 Disable animation tracking
  animation-track [-d secs] [ws-url]         Track animations
  get-animation-effect <id> [ws-url]         Get animation effect
  resolve-animation <id> [ws-url]            Resolve animation object

Canvas:
  canvas-enable [ws-url]                     Enable canvas tracking
  canvas-disable [ws-url]                    Disable canvas tracking
  get-canvas-content <canvasId> [ws-url]     Get canvas image
  start-canvas-recording <id> [frames]       Start canvas recording
  stop-canvas-recording <canvasId> [ws-url]  Stop canvas recording
  get-shader-source <progId> <type>          Get WebGL shader source

LayerTree:
  get-layer-tree <nodeId> [ws-url]           Get compositing layers
  get-compositing-reasons <layerId>          Get compositing reasons

Workers:
  worker-enable [ws-url]                     Enable worker tracking
  worker-disable [ws-url]                    Disable worker tracking
  send-to-worker <workerId> <msg> [ws-url]   Send message to worker
  get-service-worker-info [ws-url]           Get service worker info

Audit & Security:
  run-audit <testJSON> [ws-url]              Run audit
  get-certificate-info <requestId> [ws-url]  Get TLS certificate info

Browser:
  browser-extensions-enable [ws-url]         Enable browser extensions
  browser-extensions-disable [ws-url]        Disable browser extensions

Port assignment:
  ios_webkit_debug_proxy listens on port 9221 for device listing.
  Each connected device gets an incremented port starting at 9222:
    Device 1 -> localhost:9222
    Device 2 -> localhost:9223

The ws-url argument is the WebSocket URL from 'pages' output.
If omitted, connects to the first available page across all devices.

Environment:
  IWDP_WS_URL    Default WebSocket URL to connect to
  IWDP_PORT      Default device port (default: 9222)
`)
}

func getPort(args []string) int {
	if len(args) > 0 {
		if p, err := strconv.Atoi(args[0]); err == nil {
			return p
		}
	}
	if p := os.Getenv("IWDP_PORT"); p != "" {
		if port, err := strconv.Atoi(p); err == nil {
			return port
		}
	}
	return proxy.DefaultFirstDevicePort
}

func getWSURL(args []string) string {
	for _, a := range args {
		if strings.HasPrefix(a, "ws://") || strings.HasPrefix(a, "wss://") {
			return a
		}
	}
	if u := os.Getenv("IWDP_WS_URL"); u != "" {
		return u
	}
	return ""
}

func connectToPage(ctx context.Context, args []string) (*webkit.Client, error) {
	wsURL := getWSURL(args)
	if wsURL != "" {
		return webkit.NewClient(ctx, wsURL)
	}

	if !proxy.IsRunning() {
		fmt.Fprintf(os.Stderr, "ios_webkit_debug_proxy is not running.\n")
		fmt.Fprintf(os.Stderr, "Start it with: ios_webkit_debug_proxy --no-frontend\n")
		fmt.Fprintf(os.Stderr, "Install with:  brew install ios-webkit-debug-proxy\n")
		os.Exit(1)
	}

	// Auto-detect: list all devices (port 9221), then list pages on each device port.
	pages, err := proxy.ListAllPages()
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no Safari tabs found — open a page in Safari on your iOS device")
	}

	// Find first page with a WebSocket URL
	for _, p := range pages {
		if p.WebSocketDebuggerURL != "" {
			fmt.Fprintf(os.Stderr, "Connecting to: %s (%s)\n", p.Title, p.URL)
			return webkit.NewClient(ctx, p.WebSocketDebuggerURL)
		}
	}
	return nil, fmt.Errorf("no debuggable pages found — pages may already be connected to another debugger")
}

func cmdDevices(ctx context.Context) {
	if !proxy.IsRunning() {
		fmt.Fprintf(os.Stderr, "ios_webkit_debug_proxy is not running.\nStart it with: ios_webkit_debug_proxy --no-frontend\n")
		os.Exit(1)
	}
	devices, err := proxy.ListDevices()
	if err != nil {
		fatal(err)
	}
	if len(devices) == 0 {
		fmt.Println("No devices connected.")
		return
	}
	for _, d := range devices {
		fmt.Printf("%-40s  %s  (pages on %s)\n", d.DeviceName, d.DeviceID, d.URL)
	}
}

func cmdPages(ctx context.Context, args []string) {
	port := getPort(args)
	if !proxy.IsRunning() {
		fmt.Fprintf(os.Stderr, "ios_webkit_debug_proxy is not running.\nStart it with: ios_webkit_debug_proxy --no-frontend\n")
		os.Exit(1)
	}
	pages, err := proxy.ListPages(port)
	if err != nil {
		fatal(err)
	}
	if len(pages) == 0 {
		fmt.Println("No Safari tabs found.")
		return
	}
	for _, p := range pages {
		ws := p.WebSocketDebuggerURL
		if ws == "" {
			ws = "(not debuggable)"
		}
		fmt.Printf("[%d] %s\n    %s\n    %s\n", p.PageID, p.Title, p.URL, ws)
	}
}

func cmdEval(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli eval <expression> [ws-url]")
		os.Exit(1)
	}

	expr := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: closing connection: %v\n", err)
		}
	}()

	result, err := tools.EvaluateScript(ctx, client, expr, true)
	if err != nil {
		fatal(err)
	}

	if result.Result.Value != nil {
		var val interface{}
		_ = json.Unmarshal(result.Result.Value, &val)
		switch v := val.(type) {
		case string:
			fmt.Println(v)
		default:
			out, _ := json.MarshalIndent(val, "", "  ")
			fmt.Println(string(out))
		}
	} else if result.Result.Description != "" {
		fmt.Println(result.Result.Description)
	} else {
		fmt.Printf("(%s)\n", result.Result.Type)
	}
}

func cmdNavigate(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli navigate <url> [ws-url]")
		os.Exit(1)
	}

	url := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: closing connection: %v\n", err)
		}
	}()

	if err := tools.Navigate(ctx, client, url); err != nil {
		fatal(err)
	}
	fmt.Println("Navigated to", url)
}

func cmdScreenshot(ctx context.Context, args []string) {
	outputFile := "screenshot.png"
	var wsArgs []string

	for i := 0; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			outputFile = args[i+1]
			i++
		} else {
			wsArgs = append(wsArgs, args[i])
		}
	}

	client, err := connectToPage(ctx, wsArgs)
	if err != nil {
		fatal(err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: closing connection: %v\n", err)
		}
	}()

	dataURL, err := tools.TakeScreenshot(ctx, client)
	if err != nil {
		fatal(err)
	}

	// Strip data URL prefix to get raw base64
	b64Data := dataURL
	if idx := strings.Index(dataURL, ","); idx >= 0 {
		b64Data = dataURL[idx+1:]
	}

	data, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		fatal(fmt.Errorf("decoding screenshot: %w", err))
	}

	if err := os.WriteFile(outputFile, data, 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("Screenshot saved to %s (%d bytes)\n", outputFile, len(data))
}

func cmdCookies(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: closing connection: %v\n", err)
		}
	}()

	cookies, err := tools.GetCookies(ctx, client)
	if err != nil {
		fatal(err)
	}

	if len(cookies) == 0 {
		fmt.Println("No cookies.")
		return
	}

	for _, c := range cookies {
		flags := ""
		if c.HTTPOnly {
			flags += " httpOnly"
		}
		if c.Secure {
			flags += " secure"
		}
		if c.Session {
			flags += " session"
		}
		fmt.Printf("%s=%s  domain=%s path=%s%s\n", c.Name, c.Value, c.Domain, c.Path, flags)
	}
}

func cmdDOM(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: closing connection: %v\n", err)
		}
	}()

	selector := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "ws://") && !strings.HasPrefix(args[0], "wss://") {
		selector = args[0]
	}

	if selector != "" {
		// Get document root first
		doc, err := tools.GetDocument(ctx, client, 1)
		if err != nil {
			fatal(err)
		}
		nodeID, err := tools.QuerySelector(ctx, client, doc.NodeID, selector)
		if err != nil {
			fatal(err)
		}
		html, err := tools.GetOuterHTML(ctx, client, nodeID)
		if err != nil {
			fatal(err)
		}
		fmt.Println(html)
	} else {
		doc, err := tools.GetDocument(ctx, client, 3)
		if err != nil {
			fatal(err)
		}
		out, _ := json.MarshalIndent(doc, "", "  ")
		fmt.Println(string(out))
	}
}

func cmdConsole(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: closing connection: %v\n", err)
		}
	}()

	// Register a handler that prints console messages immediately as they arrive.
	client.OnEvent("Console.messageAdded", func(method string, params json.RawMessage) {
		var envelope struct {
			Message struct {
				Level  string `json:"level"`
				Text   string `json:"text"`
				Source string `json:"source"`
				URL    string `json:"url"`
				Line   int    `json:"line"`
			} `json:"message"`
		}
		if err := json.Unmarshal(params, &envelope); err != nil {
			return
		}
		m := envelope.Message
		loc := ""
		if m.URL != "" {
			loc = fmt.Sprintf(" (%s:%d)", m.URL, m.Line)
		}
		fmt.Printf("[%s] %s%s\n", m.Level, m.Text, loc)
	})

	if _, err := client.Send(ctx, "Console.enable", nil); err != nil {
		fatal(err)
	}

	fmt.Fprintln(os.Stderr, "Listening for console messages (Ctrl+C to stop)...")
	<-client.Done()
}

func cmdNetwork(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: closing connection: %v\n", err)
		}
	}()

	// Register handlers that print network events immediately as they arrive.
	client.OnEvent("Network.requestWillBeSent", func(method string, params json.RawMessage) {
		var evt struct {
			Request struct {
				Method string `json:"method"`
				URL    string `json:"url"`
			} `json:"request"`
		}
		if err := json.Unmarshal(params, &evt); err != nil {
			return
		}
		fmt.Printf("→ %s %s\n", evt.Request.Method, evt.Request.URL)
	})

	client.OnEvent("Network.responseReceived", func(method string, params json.RawMessage) {
		var evt struct {
			Response struct {
				URL        string `json:"url"`
				Status     int    `json:"status"`
				StatusText string `json:"statusText"`
				MIMEType   string `json:"mimeType"`
			} `json:"response"`
		}
		if err := json.Unmarshal(params, &evt); err != nil {
			return
		}
		fmt.Printf("← %d %s %s (%s)\n", evt.Response.Status, evt.Response.StatusText, evt.Response.URL, evt.Response.MIMEType)
	})

	client.OnEvent("Network.loadingFailed", func(method string, params json.RawMessage) {
		var evt struct {
			RequestID string `json:"requestId"`
			ErrorText string `json:"errorText"`
			Canceled  bool   `json:"canceled"`
		}
		if err := json.Unmarshal(params, &evt); err != nil {
			return
		}
		if evt.Canceled {
			fmt.Printf("✕ %s (canceled)\n", evt.RequestID)
		} else {
			fmt.Printf("✕ %s: %s\n", evt.RequestID, evt.ErrorText)
		}
	})

	if _, err := client.Send(ctx, "Network.enable", nil); err != nil {
		fatal(err)
	}

	fmt.Fprintln(os.Stderr, "Monitoring network requests (Ctrl+C to stop)...")
	<-client.Done()
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
