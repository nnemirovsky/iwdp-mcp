package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
)

func cmdQuerySelector(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli query-selector <selector> [ws-url]")
		os.Exit(1)
	}

	selector := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	doc, err := tools.GetDocument(ctx, client, 1)
	if err != nil {
		fatal(err)
	}
	nodeID, err := tools.QuerySelector(ctx, client, doc.NodeID, selector)
	if err != nil {
		fatal(err)
	}
	fmt.Println(nodeID)
}

func cmdGetOuterHTML(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-outer-html <nodeId> [ws-url]")
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

	html, err := tools.GetOuterHTML(ctx, client, nodeID)
	if err != nil {
		fatal(err)
	}
	fmt.Println(html)
}

func cmdQuerySelectorAll(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli query-selector-all <selector> [ws-url]")
		os.Exit(1)
	}

	selector := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	doc, err := tools.GetDocument(ctx, client, 1)
	if err != nil {
		fatal(err)
	}
	nodeIDs, err := tools.QuerySelectorAll(ctx, client, doc.NodeID, selector)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(nodeIDs, "", "  ")
	fmt.Println(string(out))
}

func cmdGetAttributes(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-attributes <nodeId> [ws-url]")
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

	attrs, err := tools.GetAttributes(ctx, client, nodeID)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(attrs, "", "  ")
	fmt.Println(string(out))
}

func cmdGetEventListeners(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-event-listeners <nodeId> [ws-url]")
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

	listeners, err := tools.GetEventListeners(ctx, client, nodeID)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(listeners, "", "  ")
	fmt.Println(string(out))
}

func cmdHighlightNode(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli highlight-node <nodeId> [ws-url]")
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

	if err := tools.HighlightNode(ctx, client, nodeID); err != nil {
		fatal(err)
	}
	fmt.Println("Node highlighted")
}

func cmdHideHighlight(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.HideHighlight(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Highlight hidden")
}
