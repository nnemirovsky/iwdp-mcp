package tools

import (
	"context"
	"encoding/json"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// CPUStartProfiling starts the CPU profiler.
func CPUStartProfiling(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "CPUProfiler.startTracking", nil)
	return err
}

// CPUStopProfiling stops the CPU profiler and returns the profiling result.
func CPUStopProfiling(ctx context.Context, client *webkit.Client) (json.RawMessage, error) {
	result, err := client.Send(ctx, "CPUProfiler.stopTracking", nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ScriptStartProfiling starts the script profiler with sample collection enabled.
func ScriptStartProfiling(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "ScriptProfiler.startTracking", map[string]interface{}{
		"includeSamples": true,
	})
	return err
}

// ScriptStopProfiling stops the script profiler and returns the profiling result.
func ScriptStopProfiling(ctx context.Context, client *webkit.Client) (json.RawMessage, error) {
	result, err := client.Send(ctx, "ScriptProfiler.stopTracking", nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}
