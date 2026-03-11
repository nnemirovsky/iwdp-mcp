package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
)

func cmdGetMatchedStyles(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-matched-styles <nodeId> [ws-url]")
		os.Exit(1)
	}
	nodeID, err := strconv.Atoi(args[0])
	if err != nil {
		fatal(fmt.Errorf("invalid nodeId: %s", args[0]))
	}
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	result, err := tools.GetMatchedStyles(ctx, client, nodeID)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(json.RawMessage(result), "", "  ")
	fmt.Println(string(out))
}

func cmdGetComputedStyle(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-computed-style <nodeId> [ws-url]")
		os.Exit(1)
	}
	nodeID, err := strconv.Atoi(args[0])
	if err != nil {
		fatal(fmt.Errorf("invalid nodeId: %s", args[0]))
	}
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	props, err := tools.GetComputedStyle(ctx, client, nodeID)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(props, "", "  ")
	fmt.Println(string(out))
}

func cmdGetInlineStyles(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-inline-styles <nodeId> [ws-url]")
		os.Exit(1)
	}
	nodeID, err := strconv.Atoi(args[0])
	if err != nil {
		fatal(fmt.Errorf("invalid nodeId: %s", args[0]))
	}
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	style, err := tools.GetInlineStyles(ctx, client, nodeID)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(style, "", "  ")
	fmt.Println(string(out))
}

func cmdSetStyleText(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli set-style-text <styleId-json> <text> [ws-url]")
		os.Exit(1)
	}
	var styleID json.RawMessage
	if err := json.Unmarshal([]byte(args[0]), &styleID); err != nil {
		fatal(fmt.Errorf("invalid styleId JSON: %w", err))
	}
	text := args[1]
	client, err := connectToPage(ctx, args[2:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	style, err := tools.SetStyleText(ctx, client, styleID, text)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(style, "", "  ")
	fmt.Println(string(out))
}

func cmdGetAllStylesheets(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	sheets, err := tools.GetAllStylesheets(ctx, client)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(sheets, "", "  ")
	fmt.Println(string(out))
}

func cmdGetStylesheetText(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-stylesheet-text <stylesheetId> [ws-url]")
		os.Exit(1)
	}
	sheetID := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	text, err := tools.GetStylesheetText(ctx, client, sheetID)
	if err != nil {
		fatal(err)
	}
	fmt.Println(text)
}

func cmdForcePseudoState(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli force-pseudo-state <nodeId> <state,...> [ws-url]")
		fmt.Fprintln(os.Stderr, "States: hover, focus, active, visited, focus-within, focus-visible")
		os.Exit(1)
	}
	nodeID, err := strconv.Atoi(args[0])
	if err != nil {
		fatal(fmt.Errorf("invalid nodeId: %s", args[0]))
	}
	states := strings.Split(args[1], ",")
	client, err := connectToPage(ctx, args[2:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.ForcePseudoState(ctx, client, nodeID, states); err != nil {
		fatal(err)
	}
	fmt.Printf("Forced pseudo state %v on node %d\n", states, nodeID)
}
