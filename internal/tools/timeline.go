package tools

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// TimelineStart starts timeline recording. If maxCallStackDepth > 0,
// it is included in the parameters to control stack trace depth.
func TimelineStart(ctx context.Context, client *webkit.Client, maxCallStackDepth int) error {
	var params map[string]interface{}
	if maxCallStackDepth > 0 {
		params = map[string]interface{}{
			"maxCallStackDepth": maxCallStackDepth,
		}
	}
	_, err := client.Send(ctx, "Timeline.start", params)
	return err
}

// TimelineStop stops timeline recording.
func TimelineStop(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Timeline.stop", nil)
	return err
}

// TimelineCollector collects timeline events via the Timeline.eventRecorded event.
type TimelineCollector struct {
	mu      sync.Mutex
	events  []webkit.TimelineEvent
	started bool
}

// NewTimelineCollector creates a new TimelineCollector.
func NewTimelineCollector() *TimelineCollector {
	return &TimelineCollector{}
}

// Start enables timeline recording and registers a handler for Timeline.eventRecorded events.
// If maxCallStackDepth > 0, it is included in the parameters to control stack trace depth.
// It is idempotent: calling Start multiple times will not register duplicate handlers.
func (t *TimelineCollector) Start(ctx context.Context, client *webkit.Client, maxCallStackDepth int) error {
	t.mu.Lock()
	if t.started {
		t.mu.Unlock()
		return nil
	}
	t.started = true
	t.mu.Unlock()

	client.OnEvent("Timeline.eventRecorded", func(method string, params json.RawMessage) {
		var envelope struct {
			Record webkit.TimelineEvent `json:"record"`
		}
		if err := json.Unmarshal(params, &envelope); err != nil {
			return
		}
		t.mu.Lock()
		t.events = append(t.events, envelope.Record)
		if len(t.events) > maxCollectorEntries {
			t.events = t.events[len(t.events)-maxCollectorEntries:]
		}
		t.mu.Unlock()
	})

	var p map[string]interface{}
	if maxCallStackDepth > 0 {
		p = map[string]interface{}{
			"maxCallStackDepth": maxCallStackDepth,
		}
	}
	_, err := client.Send(ctx, "Timeline.start", p)
	return err
}

// Stop stops timeline recording.
func (t *TimelineCollector) Stop(ctx context.Context, client *webkit.Client) error {
	t.mu.Lock()
	t.started = false
	t.mu.Unlock()
	_, err := client.Send(ctx, "Timeline.stop", nil)
	return err
}

// GetEvents returns a copy of the collected timeline events.
func (t *TimelineCollector) GetEvents() []webkit.TimelineEvent {
	t.mu.Lock()
	defer t.mu.Unlock()
	events := make([]webkit.TimelineEvent, len(t.events))
	copy(events, t.events)
	return events
}
