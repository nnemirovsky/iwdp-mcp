package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
)

func cmdClick(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli click <selector> [ws-url]")
		os.Exit(1)
	}
	selector := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.Click(ctx, client, selector); err != nil {
		fatal(err)
	}
	fmt.Printf("Clicked %s\n", selector)
}

func cmdFill(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli fill <selector> <value> [ws-url]")
		os.Exit(1)
	}
	selector := args[0]
	value := args[1]
	client, err := connectToPage(ctx, args[2:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.Fill(ctx, client, selector, value); err != nil {
		fatal(err)
	}
	fmt.Printf("Filled %s with %q\n", selector, value)
}

func cmdTypeText(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli type-text <text> [ws-url]")
		os.Exit(1)
	}
	text := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.TypeText(ctx, client, text); err != nil {
		fatal(err)
	}
	fmt.Println("Text typed")
}
