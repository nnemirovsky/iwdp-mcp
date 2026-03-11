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

// Animation

func cmdAnimationEnable(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.AnimationEnable(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Animation tracking enabled")
}

func cmdAnimationDisable(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.AnimationDisable(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Animation tracking disabled")
}

func cmdAnimationTrack(ctx context.Context, args []string) {
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

	collector := tools.NewAnimationTrackingCollector()
	if err := collector.Start(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Fprintf(os.Stderr, "Tracking animations for %v...\n", duration)
	time.Sleep(duration)
	result, err := collector.Stop(ctx, client)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
}

func cmdGetAnimationEffect(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-animation-effect <animationId> [ws-url]")
		os.Exit(1)
	}
	animID := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	result, err := tools.GetAnimationEffect(ctx, client, animID)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(json.RawMessage(result), "", "  ")
	fmt.Println(string(out))
}

func cmdResolveAnimation(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli resolve-animation <animationId> [ws-url]")
		os.Exit(1)
	}
	animID := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	result, err := tools.ResolveAnimation(ctx, client, animID, "cli")
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
}

// Canvas

func cmdCanvasEnable(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.CanvasEnable(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Canvas tracking enabled")
}

func cmdCanvasDisable(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.CanvasDisable(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Canvas tracking disabled")
}

func cmdGetCanvasContent(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-canvas-content <canvasId> [ws-url]")
		os.Exit(1)
	}
	canvasID := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	content, err := tools.GetCanvasContent(ctx, client, canvasID)
	if err != nil {
		fatal(err)
	}
	fmt.Println(content)
}

func cmdStartCanvasRecording(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli start-canvas-recording <canvasId> [frameCount] [ws-url]")
		os.Exit(1)
	}
	canvasID := args[0]
	frameCount := 0
	var wsArgs []string
	for i := 1; i < len(args); i++ {
		if fc, err := strconv.Atoi(args[i]); err == nil && frameCount == 0 {
			frameCount = fc
		} else {
			wsArgs = append(wsArgs, args[i])
		}
	}
	client, err := connectToPage(ctx, wsArgs)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.StartCanvasRecording(ctx, client, canvasID, frameCount); err != nil {
		fatal(err)
	}
	fmt.Println("Canvas recording started")
}

func cmdStopCanvasRecording(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli stop-canvas-recording <canvasId> [ws-url]")
		os.Exit(1)
	}
	canvasID := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.StopCanvasRecording(ctx, client, canvasID); err != nil {
		fatal(err)
	}
	fmt.Println("Canvas recording stopped")
}

func cmdGetShaderSource(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-shader-source <programId> <shaderType> [ws-url]")
		os.Exit(1)
	}
	programID := args[0]
	shaderType := args[1]
	client, err := connectToPage(ctx, args[2:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	source, err := tools.GetShaderSource(ctx, client, programID, shaderType)
	if err != nil {
		fatal(err)
	}
	fmt.Println(source)
}

// LayerTree

func cmdGetLayerTree(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-layer-tree <nodeId> [ws-url]")
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

	result, err := tools.GetLayerTree(ctx, client, nodeID)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(json.RawMessage(result), "", "  ")
	fmt.Println(string(out))
}

func cmdGetCompositingReasons(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-compositing-reasons <layerId> [ws-url]")
		os.Exit(1)
	}
	layerID := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	result, err := tools.GetCompositingReasons(ctx, client, layerID)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(json.RawMessage(result), "", "  ")
	fmt.Println(string(out))
}

// Workers

func cmdWorkerEnable(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.WorkerEnable(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Worker tracking enabled")
}

func cmdWorkerDisable(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.WorkerDisable(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Worker tracking disabled")
}

func cmdSendToWorker(ctx context.Context, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli send-to-worker <workerId> <message> [ws-url]")
		os.Exit(1)
	}
	workerID := args[0]
	message := args[1]
	client, err := connectToPage(ctx, args[2:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.SendToWorker(ctx, client, workerID, message); err != nil {
		fatal(err)
	}
	fmt.Println("Message sent to worker")
}

func cmdGetServiceWorkerInfo(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	result, err := tools.GetServiceWorkerInfo(ctx, client)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(json.RawMessage(result), "", "  ")
	fmt.Println(string(out))
}

// Audit

func cmdRunAudit(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli run-audit <testJSON> [ws-url]")
		os.Exit(1)
	}
	testStr := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	result, err := tools.RunAudit(ctx, client, testStr)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(json.RawMessage(result), "", "  ")
	fmt.Println(string(out))
}

// Browser

func cmdBrowserExtensionsEnable(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := client.Enable(ctx, "Browser"); err != nil {
		fatal(err)
	}
	fmt.Println("Browser extensions enabled")
}

func cmdBrowserExtensionsDisable(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := client.Disable(ctx, "Browser"); err != nil {
		fatal(err)
	}
	fmt.Println("Browser extensions disabled")
}

// Security

func cmdGetCertificateInfo(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: iwdp-cli get-certificate-info <requestId> [ws-url]")
		os.Exit(1)
	}
	requestID := args[0]
	client, err := connectToPage(ctx, args[1:])
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	result, err := tools.GetCertificateInfo(ctx, client, requestID)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(json.RawMessage(result), "", "  ")
	fmt.Println(string(out))
}
