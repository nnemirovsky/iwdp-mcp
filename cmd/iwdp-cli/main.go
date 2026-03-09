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
	case "devices":
		cmdDevices(ctx)
	case "pages":
		cmdPages(ctx, args)
	case "eval":
		cmdEval(ctx, args)
	case "navigate":
		cmdNavigate(ctx, args)
	case "screenshot":
		cmdScreenshot(ctx, args)
	case "cookies":
		cmdCookies(ctx, args)
	case "dom":
		cmdDOM(ctx, args)
	case "console":
		cmdConsole(ctx, args)
	case "network":
		cmdNetwork(ctx, args)
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

Commands:
  devices                     List connected iOS devices (via iwdp port 9221)
  pages [port]                List open Safari tabs on a device port
                              (default: 9222 = first device; use 'devices' to find ports)
  eval <expression> [ws-url]  Evaluate JavaScript
  navigate <url> [ws-url]     Navigate to URL
  screenshot [-o file] [ws-url]  Take screenshot
  cookies [ws-url]            Show all cookies (incl. httpOnly)
  dom [selector] [ws-url]     Inspect DOM
  console [ws-url]            Show console messages
  network [ws-url]            Show network requests

Port assignment:
  ios_webkit_debug_proxy listens on port 9221 for device listing.
  Each connected device gets an incremented port starting at 9222:
    Device 1 → localhost:9222
    Device 2 → localhost:9223
    ...

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
