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

func cmdTimelineRecord(ctx context.Context, args []string) {
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

	collector := tools.NewTimelineCollector()
	if err := collector.Start(ctx, client, 5); err != nil {
		fatal(err)
	}
	fmt.Fprintf(os.Stderr, "Recording timeline for %v...\n", duration)
	time.Sleep(duration)
	if err := collector.Stop(ctx, client); err != nil {
		fatal(err)
	}
	events := collector.GetEvents()
	out, _ := json.MarshalIndent(events, "", "  ")
	fmt.Println(string(out))
}

func cmdMemoryTrack(ctx context.Context, args []string) {
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

	collector := tools.NewMemoryTrackingCollector()
	if err := collector.Start(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Fprintf(os.Stderr, "Tracking memory for %v...\n", duration)
	time.Sleep(duration)
	result, err := collector.Stop(ctx, client)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
}

func cmdHeapSnapshot(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	snapshot, err := tools.HeapSnapshot(ctx, client)
	if err != nil {
		fatal(err)
	}
	fmt.Println(snapshot)
}

func cmdHeapTrack(ctx context.Context, args []string) {
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

	collector := tools.NewHeapTrackingCollector()
	if err := collector.Start(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Fprintf(os.Stderr, "Tracking heap allocations for %v...\n", duration)
	time.Sleep(duration)
	result, err := collector.Stop(ctx, client)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
}

func cmdHeapGC(ctx context.Context, args []string) {
	client, err := connectToPage(ctx, args)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = client.Close() }()

	if err := tools.HeapGC(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Println("Garbage collection triggered")
}

func cmdCPUProfile(ctx context.Context, args []string) {
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

	collector := tools.NewCPUProfilerCollector()
	if err := collector.Start(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Fprintf(os.Stderr, "CPU profiling for %v...\n", duration)
	time.Sleep(duration)
	result, err := collector.Stop(ctx, client)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
}

func cmdScriptProfile(ctx context.Context, args []string) {
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

	collector := tools.NewScriptProfilerCollector()
	if err := collector.Start(ctx, client); err != nil {
		fatal(err)
	}
	fmt.Fprintf(os.Stderr, "Script profiling for %v...\n", duration)
	time.Sleep(duration)
	result, err := collector.Stop(ctx, client)
	if err != nil {
		fatal(err)
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
}
