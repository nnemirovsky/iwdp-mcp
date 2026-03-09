package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// HeapSnapshotTimeout is the maximum time to wait for a heap snapshot.
// Large pages can take a long time to serialize their heap.
var HeapSnapshotTimeout = 5 * time.Minute

// --- Memory Tracking Collector ---

// MemoryCategory represents a single memory category entry.
type MemoryCategory struct {
	Type string `json:"type"`
	Size int64  `json:"size"`
}

// MemoryTrackingEvent represents a single Memory tracking sample.
type MemoryTrackingEvent struct {
	Timestamp  float64          `json:"timestamp"`
	Categories []MemoryCategory `json:"categories"`
}

// MemoryTrackingResult holds the collected memory tracking data.
type MemoryTrackingResult struct {
	Events []MemoryTrackingEvent `json:"events"`
}

// MemoryTrackingCollector collects Memory tracking events.
type MemoryTrackingCollector struct {
	mu      sync.Mutex
	events  []MemoryTrackingEvent
	started bool
	done    chan struct{}
}

// NewMemoryTrackingCollector creates a new MemoryTrackingCollector.
func NewMemoryTrackingCollector() *MemoryTrackingCollector {
	return &MemoryTrackingCollector{}
}

// Start begins memory tracking, collecting trackingUpdate events.
func (c *MemoryTrackingCollector) Start(ctx context.Context, client *webkit.Client) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}
	c.started = true
	c.events = nil
	c.done = make(chan struct{})
	c.mu.Unlock()

	client.OnEvent("Memory.trackingUpdate", func(method string, params json.RawMessage) {
		var evt struct {
			Event MemoryTrackingEvent `json:"event"`
		}
		if err := json.Unmarshal(params, &evt); err != nil {
			return
		}
		c.mu.Lock()
		c.events = append(c.events, evt.Event)
		if len(c.events) > maxCollectorEntries {
			c.events = c.events[len(c.events)-maxCollectorEntries:]
		}
		c.mu.Unlock()
	})

	client.OnEvent("Memory.trackingComplete", func(method string, params json.RawMessage) {
		c.mu.Lock()
		ch := c.done
		c.mu.Unlock()
		if ch != nil {
			select {
			case <-ch:
			default:
				close(ch)
			}
		}
	})

	_, err := client.Send(ctx, "Memory.startTracking", nil)
	return err
}

// Stop stops memory tracking and returns the collected events.
func (c *MemoryTrackingCollector) Stop(ctx context.Context, client *webkit.Client) (*MemoryTrackingResult, error) {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return &MemoryTrackingResult{}, nil
	}
	ch := c.done
	c.mu.Unlock()

	_, err := client.Send(ctx, "Memory.stopTracking", nil)
	if err != nil {
		return nil, err
	}

	// Wait for trackingComplete event.
	if ch != nil {
		select {
		case <-ch:
		case <-ctx.Done():
		}
	}

	c.mu.Lock()
	c.started = false
	result := &MemoryTrackingResult{
		Events: make([]MemoryTrackingEvent, len(c.events)),
	}
	copy(result.Events, c.events)
	c.events = nil
	c.mu.Unlock()

	return result, nil
}

// --- Heap Snapshot ---

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

// --- Heap Tracking Collector ---

// GarbageCollection represents a GC event from Heap.garbageCollected.
type GarbageCollection struct {
	Type      string  `json:"type"` // full, partial
	StartTime float64 `json:"startTime"`
	EndTime   float64 `json:"endTime"`
}

// HeapTrackingReadyTimeout is how long Start waits for the trackingStart event
// to confirm the event pipeline is healthy. Default 5s.
var HeapTrackingReadyTimeout = 5 * time.Second

// HeapTrackingResult holds the collected heap tracking data.
type HeapTrackingResult struct {
	GCEvents        []GarbageCollection `json:"gcEvents,omitempty"`
	PipelineHealthy bool                `json:"pipelineHealthy"`
}

// HeapTrackingCollector collects Heap GC events during tracking.
//
// Heap.startTracking triggers a trackingStart event carrying a full heap
// snapshot (50-200MB+). If iwdp can relay this massive event successfully,
// the event pipeline is healthy and subsequent garbageCollected events will
// arrive. If not, the pipeline is broken and no events will be captured.
// Start() waits for the trackingStart event to diagnose this.
type HeapTrackingCollector struct {
	mu              sync.Mutex
	gcEvents        []GarbageCollection
	started         bool
	ready           chan struct{}
	pipelineHealthy bool
}

// NewHeapTrackingCollector creates a new HeapTrackingCollector.
func NewHeapTrackingCollector() *HeapTrackingCollector {
	return &HeapTrackingCollector{}
}

// PipelineHealthy reports whether the event pipeline survived the massive
// trackingStart event. If false, garbageCollected events won't arrive.
func (c *HeapTrackingCollector) PipelineHealthy() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pipelineHealthy
}

// Start begins heap tracking, collecting garbageCollected events.
// It waits up to HeapTrackingReadyTimeout for the trackingStart event to
// confirm the event pipeline is healthy.
func (c *HeapTrackingCollector) Start(ctx context.Context, client *webkit.Client) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}
	c.started = true
	c.gcEvents = nil
	c.ready = make(chan struct{})
	c.pipelineHealthy = false
	c.mu.Unlock()

	// trackingStart handler: signals that the massive snapshot event arrived
	// at our client, confirming the event pipeline is healthy. We discard
	// the snapshot data (50-200MB+) — use heap_snapshot for snapshots.
	client.OnEvent("Heap.trackingStart", func(method string, params json.RawMessage) {
		c.mu.Lock()
		c.pipelineHealthy = true
		ch := c.ready
		c.mu.Unlock()
		if ch != nil {
			select {
			case <-ch:
			default:
				close(ch)
			}
		}
	})

	client.OnEvent("Heap.garbageCollected", func(method string, params json.RawMessage) {
		var evt struct {
			Collection GarbageCollection `json:"collection"`
		}
		if json.Unmarshal(params, &evt) == nil {
			c.mu.Lock()
			c.gcEvents = append(c.gcEvents, evt.Collection)
			c.mu.Unlock()
		}
	})

	// Enable Heap domain — required for events to be dispatched.
	_, _ = client.Send(ctx, "Heap.enable", nil)

	// Pre-GC: reduce heap size before startTracking to minimize the snapshot
	// payload in the trackingStart event. A smaller snapshot is more likely
	// to survive iwdp's WebSocket relay (which has a 64MB message limit).
	_, _ = client.Send(ctx, "Heap.gc", nil)

	_, err := client.Send(ctx, "Heap.startTracking", nil)
	if err != nil {
		return err
	}

	// Wait for trackingStart event to confirm the event pipeline survived
	// the massive snapshot payload. If it arrives, GC events will work.
	// If not, iwdp couldn't relay the 50-200MB+ event and the pipeline is broken.
	c.mu.Lock()
	ch := c.ready
	c.mu.Unlock()
	select {
	case <-ch:
		// Pipeline healthy — trackingStart event arrived.
	case <-time.After(HeapTrackingReadyTimeout):
		// Timeout — event pipeline likely broken by massive snapshot.
	case <-ctx.Done():
	}

	return nil
}

// Stop stops heap tracking and returns collected GC events.
// Errors from Heap.stopTracking are swallowed because the massive snapshot
// events from trackingComplete may have already crashed the connection.
func (c *HeapTrackingCollector) Stop(ctx context.Context, client *webkit.Client) (*HeapTrackingResult, error) {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return &HeapTrackingResult{}, nil
	}
	healthy := c.pipelineHealthy
	c.mu.Unlock()

	// stopTracking triggers trackingComplete with a massive snapshot payload
	// that often crashes iwdp. We send it but don't fail if it errors.
	_, _ = client.Send(ctx, "Heap.stopTracking", nil)

	c.mu.Lock()
	c.started = false
	result := &HeapTrackingResult{
		GCEvents:        make([]GarbageCollection, len(c.gcEvents)),
		PipelineHealthy: healthy,
	}
	copy(result.GCEvents, c.gcEvents)
	c.gcEvents = nil
	c.mu.Unlock()

	return result, nil
}

// HeapGC triggers garbage collection.
func HeapGC(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Heap.gc", nil)
	return err
}
