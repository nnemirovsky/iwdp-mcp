package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
)

func cmdGetResponseBody(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-response-body <requestId> [ws-url]")
		os.Exit(1)
	}
	requestID := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	body, base64Encoded, err := tools.GetResponseBody(ctx, client, requestID)
	if err != nil {
		fatal(err)
	}
	if base64Encoded {
		fmt.Printf("[base64 encoded, %d bytes]\n", len(body))
	}
	fmt.Println(body)
}

func cmdSetExtraHeaders(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli set-extra-headers '{\"Header\": \"Value\"}' [ws-url]")
		os.Exit(1)
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(args[0]), &headers); err != nil {
		fatal(fmt.Errorf("invalid headers JSON: %w", err))
	}
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.SetExtraHeaders(ctx, client, headers); err != nil {
		fatal(err)
	}
	fmt.Println("Extra headers set")
}

func cmdSetRequestInterception(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli set-request-interception <enabled:true/false> [ws-url]")
		os.Exit(1)
	}
	enabled := args[0] == "true"
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.SetRequestInterception(ctx, client, enabled); err != nil {
		fatal(err)
	}
	fmt.Printf("Request interception: %v\n", enabled)
}

func cmdInterceptContinue(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli intercept-continue <requestId> [--stage request|response] [ws-url]")
		os.Exit(1)
	}
	requestID := args[0]
	stage := "request"
	var wsArgs []string
	for i := 1; i < len(args); i++ {
		if args[i] == "--stage" && i+1 < len(args) {
			stage = args[i+1]
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

	if err := tools.InterceptContinue(ctx, client, requestID, stage); err != nil {
		fatal(err)
	}
	fmt.Println("Request continued")
}

func cmdInterceptWithResponse(ctx context.Context, args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli intercept-respond <requestId> <statusCode> <body> [--stage request] [ws-url]")
		os.Exit(1)
	}
	requestID := args[0]
	statusCode, err := strconv.Atoi(args[1])
	if err != nil {
		fatal(fmt.Errorf("invalid status code: %s", args[1]))
	}
	body := args[2]
	stage := "request"
	var wsArgs []string
	for i := 3; i < len(args); i++ {
		if args[i] == "--stage" && i+1 < len(args) {
			stage = args[i+1]
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

	if err := tools.InterceptWithResponse(ctx, client, requestID, stage, statusCode, nil, body, false); err != nil {
		fatal(err)
	}
	fmt.Printf("Responded with %d\n", statusCode)
}

func cmdSetEmulatedConditions(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli set-network-conditions <bytesPerSecondLimit> [ws-url]")
		os.Exit(1)
	}
	limit, err := strconv.Atoi(args[0])
	if err != nil {
		fatal(fmt.Errorf("invalid limit: %s", args[0]))
	}
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.SetEmulatedConditions(ctx, client, limit); err != nil {
		fatal(err)
	}
	fmt.Printf("Network throttled to %d bytes/sec\n", limit)
}

func cmdSetResourceCachingDisabled(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli disable-cache <true/false> [ws-url]")
		os.Exit(1)
	}
	disabled := args[0] == "true"
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.SetResourceCachingDisabled(ctx, client, disabled); err != nil {
		fatal(err)
	}
	fmt.Printf("Resource caching disabled: %v\n", disabled)
}
