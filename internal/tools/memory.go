package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// HeapSnapshotTimeout is the maximum time to wait for a heap snapshot.
// Large pages can take a long time to serialize their heap.
var HeapSnapshotTimeout = 5 * time.Minute

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

// HeapSnapshot takes a heap snapshot and saves it to a temp file.
// Heap snapshots can be 50-200+ MB on heavy pages, so we stream directly
// to disk instead of holding the entire snapshot in memory.
func HeapSnapshot(ctx context.Context, client *webkit.Client) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, HeapSnapshotTimeout)
	defer cancel()

	result, err := client.Send(ctx, "Heap.snapshot", nil)
	if err != nil {
		return "", fmt.Errorf("heap snapshot failed (page may be too large): %w", err)
	}

	tmpDir := filepath.Join(os.TempDir(), "iwdp-mcp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}
	f, err := os.CreateTemp(tmpDir, "heap-snapshot-*.json")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Extract snapshotData string if present, otherwise write raw result
	var snap struct {
		SnapshotData json.RawMessage `json:"snapshotData"`
	}
	if json.Unmarshal(result, &snap) == nil && len(snap.SnapshotData) > 0 {
		// snapshotData is a JSON string — unquote it to get the raw snapshot
		var data string
		if json.Unmarshal(snap.SnapshotData, &data) == nil {
			if _, err := f.WriteString(data); err != nil {
				return "", fmt.Errorf("writing snapshot: %w", err)
			}
		} else {
			// Not a string, write as-is
			if _, err := f.Write(snap.SnapshotData); err != nil {
				return "", fmt.Errorf("writing snapshot: %w", err)
			}
		}
	} else {
		if _, err := f.Write(result); err != nil {
			return "", fmt.Errorf("writing snapshot: %w", err)
		}
	}

	return f.Name(), nil
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
