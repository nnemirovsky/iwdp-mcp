package tools

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// --- CPU Profiler ---

// CPUProfileEvent represents a single CPU usage sample.
type CPUProfileEvent struct {
	Timestamp float64           `json:"timestamp"`
	Usage     float64           `json:"usage"`
	Threads   []json.RawMessage `json:"threads,omitempty"`
}

// CPUProfileResult holds the collected CPU profiling data.
type CPUProfileResult struct {
	Events []CPUProfileEvent `json:"events"`
}

// CPUProfilerCollector collects CPU profiling events.
type CPUProfilerCollector struct {
	mu      sync.Mutex
	events  []CPUProfileEvent
	started bool
	done    chan struct{}
}

// NewCPUProfilerCollector creates a new CPUProfilerCollector.
func NewCPUProfilerCollector() *CPUProfilerCollector {
	return &CPUProfilerCollector{}
}

// Start begins CPU profiling, collecting trackingUpdate events.
func (c *CPUProfilerCollector) Start(ctx context.Context, client *webkit.Client) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}
	c.started = true
	c.events = nil
	c.done = make(chan struct{})
	c.mu.Unlock()

	client.OnEvent("CPUProfiler.trackingUpdate", func(method string, params json.RawMessage) {
		var evt struct {
			Event CPUProfileEvent `json:"event"`
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

	client.OnEvent("CPUProfiler.trackingComplete", func(method string, params json.RawMessage) {
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

	_, err := client.Send(ctx, "CPUProfiler.startTracking", nil)
	return err
}

// Stop stops CPU profiling and returns the collected events.
func (c *CPUProfilerCollector) Stop(ctx context.Context, client *webkit.Client) (*CPUProfileResult, error) {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return &CPUProfileResult{}, nil
	}
	ch := c.done
	c.mu.Unlock()

	_, err := client.Send(ctx, "CPUProfiler.stopTracking", nil)
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
	result := &CPUProfileResult{
		Events: make([]CPUProfileEvent, len(c.events)),
	}
	copy(result.Events, c.events)
	c.events = nil
	c.mu.Unlock()

	return result, nil
}

// --- Script Profiler ---

// ScriptProfileEvent represents a script execution event.
type ScriptProfileEvent struct {
	StartTime float64 `json:"startTime"`
	EndTime   float64 `json:"endTime"`
	Type      string  `json:"type"`
}

// ScriptProfileResult holds the collected script profiling data.
type ScriptProfileResult struct {
	Events  []ScriptProfileEvent `json:"events"`
	Samples json.RawMessage      `json:"samples,omitempty"`
}

// ScriptProfilerCollector collects script profiling events.
type ScriptProfilerCollector struct {
	mu      sync.Mutex
	events  []ScriptProfileEvent
	samples json.RawMessage
	started bool
	done    chan struct{}
}

// NewScriptProfilerCollector creates a new ScriptProfilerCollector.
func NewScriptProfilerCollector() *ScriptProfilerCollector {
	return &ScriptProfilerCollector{}
}

// Start begins script profiling with sample collection.
func (c *ScriptProfilerCollector) Start(ctx context.Context, client *webkit.Client) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}
	c.started = true
	c.events = nil
	c.samples = nil
	c.done = make(chan struct{})
	c.mu.Unlock()

	client.OnEvent("ScriptProfiler.trackingUpdate", func(method string, params json.RawMessage) {
		var evt struct {
			Event ScriptProfileEvent `json:"event"`
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

	client.OnEvent("ScriptProfiler.trackingComplete", func(method string, params json.RawMessage) {
		// trackingComplete carries samples when includeSamples was true.
		var evt struct {
			Samples json.RawMessage `json:"samples"`
		}
		if err := json.Unmarshal(params, &evt); err == nil && len(evt.Samples) > 0 {
			c.mu.Lock()
			c.samples = evt.Samples
			c.mu.Unlock()
		}
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

	_, err := client.Send(ctx, "ScriptProfiler.startTracking", map[string]interface{}{
		"includeSamples": true,
	})
	return err
}

// Stop stops script profiling and returns the collected events and samples.
func (c *ScriptProfilerCollector) Stop(ctx context.Context, client *webkit.Client) (*ScriptProfileResult, error) {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return &ScriptProfileResult{}, nil
	}
	ch := c.done
	c.mu.Unlock()

	_, err := client.Send(ctx, "ScriptProfiler.stopTracking", nil)
	if err != nil {
		return nil, err
	}

	// Wait for trackingComplete event (carries samples).
	if ch != nil {
		select {
		case <-ch:
		case <-ctx.Done():
		}
	}

	c.mu.Lock()
	c.started = false
	result := &ScriptProfileResult{
		Events:  make([]ScriptProfileEvent, len(c.events)),
		Samples: c.samples,
	}
	copy(result.Events, c.events)
	c.events = nil
	c.samples = nil
	c.mu.Unlock()

	return result, nil
}
