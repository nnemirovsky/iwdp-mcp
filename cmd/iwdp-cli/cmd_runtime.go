package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
)

func cmdCallFunction(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli call-function <objectId> <functionDeclaration> [ws-url]")
		os.Exit(1)
	}

	objectID := args[0]
	funcDecl := args[1]
	client, err := connectToPage(ctx, args[2:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	result, err := tools.CallFunctionOn(ctx, client, objectID, funcDecl, nil, true)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
}

func cmdGetProperties(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-properties <objectId> [ws-url]")
		os.Exit(1)
	}

	objectID := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	props, err := tools.GetProperties(ctx, client, objectID, true)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(props, "", "  ")
	fmt.Println(string(out))
}
