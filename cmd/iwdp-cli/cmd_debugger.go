package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

func cmdDebuggerEnable(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.DebuggerEnable(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Debugger enabled")
}

func cmdDebuggerDisable(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.DebuggerDisable(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Debugger disabled")
}

func cmdSetBreakpoint(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli set-breakpoint <url> <line> [column] [--condition expr] [ws-url]")
		os.Exit(1)
	}
	bpURL := args[0]
	line, err := strconv.Atoi(args[1])
	if err != nil {
		fatal(fmt.Errorf("invalid line: %s", args[1]))
	}
	var col *int
	condition := ""
	var wsArgs []string
	for i := 2; i < len(args); i++ {
		if args[i] == "--condition" && i+1 < len(args) {
			condition = args[i+1]
			i++
		} else if c, err := strconv.Atoi(args[i]); err == nil && col == nil {
			col = &c
		} else {
			wsArgs = append(wsArgs, args[i])
		}
	}
	client, err := connectToPage(ctx, wsArgs)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	bpID, locations, err := tools.SetBreakpointByURL(ctx, client, bpURL, line, col, condition)
	if err != nil {
		fatal(err)
	}
	result := struct {
		BreakpointID webkit.BreakpointID `json:"breakpointId"`
		Locations    []webkit.Location   `json:"locations"`
	}{bpID, locations}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
}

func cmdRemoveBreakpoint(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli remove-breakpoint <breakpointId> [ws-url]")
		os.Exit(1)
	}
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.RemoveBreakpoint(ctx, client, webkit.BreakpointID(args[0])); err != nil {
		fatal(err)
	}
	fmt.Println("Breakpoint removed")
}

func cmdPause(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.Pause(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Paused")
}

func cmdResume(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.Resume(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Resumed")
}

func cmdStepOver(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.StepOver(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Stepped over")
}

func cmdStepInto(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.StepInto(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Stepped into")
}

func cmdStepOut(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.StepOut(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Stepped out")
}

func cmdGetScriptSource(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-script-source <scriptId> [ws-url]")
		os.Exit(1)
	}
	scriptID := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	source, err := tools.GetScriptSource(ctx, client, scriptID)
	if err != nil {
		fatal(err)
	}
	fmt.Println(source)
}

func cmdSearchInContent(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli search-in-content <scriptId> <query> [--caseSensitive] [--regex] [ws-url]")
		os.Exit(1)
	}
	scriptID := args[0]
	query := args[1]
	caseSensitive := false
	isRegex := false
	var wsArgs []string
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--caseSensitive":
			caseSensitive = true
		case "--regex":
			isRegex = true
		default:
			wsArgs = append(wsArgs, args[i])
		}
	}
	client, err := connectToPage(ctx, wsArgs)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	result, err := tools.SearchInContent(ctx, client, scriptID, query, caseSensitive, isRegex)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(json.RawMessage(result), "", "  ")
	fmt.Println(string(out))
}

func cmdEvalOnFrame(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli eval-on-frame <callFrameId> <expression> [ws-url]")
		os.Exit(1)
	}
	frameID := args[0]
	expr := args[1]
	client, err := connectToPage(ctx, args[2:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	result, err := tools.EvaluateOnCallFrame(ctx, client, frameID, expr, true)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
}

func cmdSetPauseOnExceptions(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli set-pause-on-exceptions <none|uncaught|all> [ws-url]")
		os.Exit(1)
	}
	state := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.SetPauseOnExceptions(ctx, client, state); err != nil {
		fatal(err)
	}
	fmt.Printf("Pause on exceptions: %s\n", state)
}

func cmdSetDOMBreakpoint(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli set-dom-breakpoint <nodeId> <type> [ws-url]")
		fmt.Fprintln(os.Stderr, "Types: subtree-modified, attribute-modified, node-removed")
		os.Exit(1)
	}
	nodeID, err := strconv.Atoi(args[0])
	if err != nil {
		fatal(fmt.Errorf("invalid nodeId: %s", args[0]))
	}
	client, err := connectToPage(ctx, args[2:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.SetDOMBreakpoint(ctx, client, nodeID, args[1]); err != nil {
		fatal(err)
	}
	fmt.Printf("DOM breakpoint set: %s on node %d\n", args[1], nodeID)
}

func cmdRemoveDOMBreakpoint(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli remove-dom-breakpoint <nodeId> <type> [ws-url]")
		os.Exit(1)
	}
	nodeID, err := strconv.Atoi(args[0])
	if err != nil {
		fatal(fmt.Errorf("invalid nodeId: %s", args[0]))
	}
	client, err := connectToPage(ctx, args[2:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.RemoveDOMBreakpoint(ctx, client, nodeID, args[1]); err != nil {
		fatal(err)
	}
	fmt.Println("DOM breakpoint removed")
}

func cmdSetEventBreakpoint(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli set-event-breakpoint <eventName> [--type listener|instrumentation] [ws-url]")
		os.Exit(1)
	}
	eventName := args[0]
	bpType := "listener"
	var wsArgs []string
	for i := 1; i < len(args); i++ {
		if args[i] == "--type" && i+1 < len(args) {
			bpType = args[i+1]
			i++
		} else {
			wsArgs = append(wsArgs, args[i])
		}
	}
	client, err := connectToPage(ctx, wsArgs)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.SetEventBreakpoint(ctx, client, bpType, eventName); err != nil {
		fatal(err)
	}
	fmt.Printf("Event breakpoint set: %s (%s)\n", eventName, bpType)
}

func cmdRemoveEventBreakpoint(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli remove-event-breakpoint <eventName> [--type listener|instrumentation] [ws-url]")
		os.Exit(1)
	}
	eventName := args[0]
	bpType := "listener"
	var wsArgs []string
	for i := 1; i < len(args); i++ {
		if args[i] == "--type" && i+1 < len(args) {
			bpType = args[i+1]
			i++
		} else {
			wsArgs = append(wsArgs, args[i])
		}
	}
	client, err := connectToPage(ctx, wsArgs)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.RemoveEventBreakpoint(ctx, client, bpType, eventName); err != nil {
		fatal(err)
	}
	fmt.Println("Event breakpoint removed")
}

func cmdSetURLBreakpoint(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli set-url-breakpoint <url> [--regex] [ws-url]")
		os.Exit(1)
	}
	bpURL := args[0]
	isRegex := false
	var wsArgs []string
	for i := 1; i < len(args); i++ {
		if args[i] == "--regex" {
			isRegex = true
		} else {
			wsArgs = append(wsArgs, args[i])
		}
	}
	client, err := connectToPage(ctx, wsArgs)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.SetURLBreakpoint(ctx, client, bpURL, isRegex); err != nil {
		fatal(err)
	}
	fmt.Printf("URL breakpoint set: %s\n", bpURL)
}

func cmdRemoveURLBreakpoint(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli remove-url-breakpoint <url> [ws-url]")
		os.Exit(1)
	}
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.RemoveURLBreakpoint(ctx, client, args[0]); err != nil {
		fatal(err)
	}
	fmt.Println("URL breakpoint removed")
}
