package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
)

func cmdConsoleMessages(ctx context.Context, args []string) {
	duration := 3 * time.Second
	var wsArgs []string
	for i := 0; i < len(args); i++ {
		if args[i] == "-d" && i+1 < len(args) {
			secs, err := strconv.Atoi(args[i+1])
			if err == nil {
				duration = time.Duration(secs) * time.Second
			}
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

	collector := tools.NewConsoleCollector()
	if err := collector.Start(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Fprintf(os.Stderr, "Collecting console messages for %v...\n", duration)
	time.Sleep(duration)
	if err := collector.Stop(ctx, client); err != nil {
		fatal(err)
	}
	msgs := collector.GetMessages()
	out, _ := json.MarshalIndent(msgs, "", "  ")
	fmt.Println(string(out))
}

func cmdClearConsole(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.ClearConsoleMessages(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Console cleared")
}

func cmdSetLogLevel(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli set-log-level <source> <level> [ws-url]")
		os.Exit(1)
	}
	source := args[0]
	level := args[1]
	client, err := connectToPage(ctx, args[2:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.SetLogLevel(ctx, client, source, level); err != nil {
		fatal(err)
	}
	fmt.Printf("Log level set: %s = %s\n", source, level)
}
