package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/nnemirovsky/iwdp-mcp/internal/proxy"
	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
)

func cmdRestartIWDP(ctx context.Context) {
	if err := proxy.Restart(ctx); err != nil {
		fatal(err)
	}
	fmt.Println("ios_webkit_debug_proxy restarted")
}

func cmdStatus(_ context.Context) {
	if proxy.IsRunning() {
		fmt.Println("ios_webkit_debug_proxy is running")
	} else {
		fmt.Println("ios_webkit_debug_proxy is NOT running")
		fmt.Fprintln(os.Stderr, "Start it with: ios_webkit_debug_proxy --no-frontend")
		os.Exit(1)
	}
}

func cmdReload(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	ignoreCache := false
	for _, a := range args {
		if a == "--ignore-cache" {
			ignoreCache = true
		}
	}
	if err := tools.Reload(ctx, client, ignoreCache); err != nil {
		fatal(err)
	}
	fmt.Println("Page reloaded")
}

func cmdSnapshotNode(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli snapshot-node <nodeId> [-o file] [ws-url]")
		os.Exit(1)
	}

	nodeID, err := strconv.Atoi(args[0])
	if err != nil {
		fatal(fmt.Errorf("invalid nodeId: %s", args[0]))
	}

	outputFile := "snapshot.png"
	var wsArgs []string
	for i := 1; i < len(args); i++ {
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
	defer func() { _ = client.Close() }()

	dataURL, err := tools.SnapshotNode(ctx, client, nodeID)
	if err != nil {
		fatal(err)
	}

	b64Data := dataURL
	if idx := strings.Index(dataURL, ","); idx >= 0 {
		b64Data = dataURL[idx+1:]
	}
	data, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		fatal(fmt.Errorf("decoding snapshot: %w", err))
	}
	if err := os.WriteFile(outputFile, data, 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("Snapshot saved to %s (%d bytes)\n", outputFile, len(data))
}
