package tools

import (
	"context"
	"encoding/json"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// MemoryStartTracking starts memory tracking.
func MemoryStartTracking(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Memory.startTracking", nil)
	return err
}

// MemoryStopTracking stops memory tracking.
func MemoryStopTracking(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Memory.stopTracking", nil)
	return err
}

// HeapSnapshot takes a heap snapshot and returns the raw result.
func HeapSnapshot(ctx context.Context, client *webkit.Client) (json.RawMessage, error) {
	result, err := client.Send(ctx, "Heap.snapshot", nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// HeapStartTracking starts heap tracking.
func HeapStartTracking(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Heap.startTracking", nil)
	return err
}

// HeapStopTracking stops heap tracking.
func HeapStopTracking(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Heap.stopTracking", nil)
	return err
}

// HeapGC triggers garbage collection.
func HeapGC(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Heap.gc", nil)
	return err
}
