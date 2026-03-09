package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit/testutil"
)

// Ensure imports are used.
var (
	_ *testutil.MockServer
	_ webkit.BreakpointID
)

// ---------------------------------------------------------------------------
// Debugger domain
// ---------------------------------------------------------------------------

func TestDebuggerEnable(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Debugger.enable", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.DebuggerEnable(ctx, client); err != nil {
		t.Fatalf("DebuggerEnable returned error: %v", err)
	}
}

func TestDebuggerDisable(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Debugger.disable", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.DebuggerDisable(ctx, client); err != nil {
		t.Fatalf("DebuggerDisable returned error: %v", err)
	}
}

func TestSetBreakpointByURL(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Debugger.setBreakpointByUrl", map[string]interface{}{
		"breakpointId": "bp-1",
		"locations": []map[string]interface{}{
			{"scriptId": "1", "lineNumber": 10, "columnNumber": 0},
		},
	})

	ctx := context.Background()
	bpID, locations, err := tools.SetBreakpointByURL(ctx, client, "test.js", 10, nil, "")
	if err != nil {
		t.Fatalf("SetBreakpointByURL returned error: %v", err)
	}
	if string(bpID) != "bp-1" {
		t.Errorf("expected breakpointId %q, got %q", "bp-1", bpID)
	}
	if len(locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(locations))
	}
	if locations[0].ScriptID != "1" {
		t.Errorf("expected scriptId %q, got %q", "1", locations[0].ScriptID)
	}
	if locations[0].LineNumber != 10 {
		t.Errorf("expected lineNumber 10, got %d", locations[0].LineNumber)
	}
}

func TestSetBreakpointByURLWithCondition(t *testing.T) {
	mock, client := setup(t)

	var receivedColumn interface{}
	var receivedCondition string
	mock.Handle("Debugger.setBreakpointByUrl", func(_ string, params json.RawMessage) (interface{}, error) {
		var p map[string]interface{}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedColumn = p["columnNumber"]
		if opts, ok := p["options"].(map[string]interface{}); ok {
			if c, ok := opts["condition"].(string); ok {
				receivedCondition = c
			}
		}
		return map[string]interface{}{
			"breakpointId": "bp-2",
			"locations": []map[string]interface{}{
				{"scriptId": "1", "lineNumber": 10, "columnNumber": 5},
			},
		}, nil
	})

	ctx := context.Background()
	col := 5
	bpID, _, err := tools.SetBreakpointByURL(ctx, client, "test.js", 10, &col, "x > 10")
	if err != nil {
		t.Fatalf("SetBreakpointByURL returned error: %v", err)
	}
	if string(bpID) != "bp-2" {
		t.Errorf("expected breakpointId %q, got %q", "bp-2", bpID)
	}
	if receivedColumn == nil {
		t.Errorf("expected columnNumber to be sent, but it was nil")
	} else if int(receivedColumn.(float64)) != 5 {
		t.Errorf("expected columnNumber 5, got %v", receivedColumn)
	}
	if receivedCondition != "x > 10" {
		t.Errorf("expected condition %q, got %q", "x > 10", receivedCondition)
	}
}

func TestRemoveBreakpoint(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Debugger.removeBreakpoint", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.RemoveBreakpoint(ctx, client, "bp-1"); err != nil {
		t.Fatalf("RemoveBreakpoint returned error: %v", err)
	}
}

func TestPause(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Debugger.pause", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.Pause(ctx, client); err != nil {
		t.Fatalf("Pause returned error: %v", err)
	}
}

func TestResume(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Debugger.resume", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.Resume(ctx, client); err != nil {
		t.Fatalf("Resume returned error: %v", err)
	}
}

func TestStepOver(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Debugger.stepOver", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.StepOver(ctx, client); err != nil {
		t.Fatalf("StepOver returned error: %v", err)
	}
}

func TestStepInto(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Debugger.stepInto", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.StepInto(ctx, client); err != nil {
		t.Fatalf("StepInto returned error: %v", err)
	}
}

func TestStepOut(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Debugger.stepOut", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.StepOut(ctx, client); err != nil {
		t.Fatalf("StepOut returned error: %v", err)
	}
}

func TestGetScriptSource(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Debugger.getScriptSource", map[string]interface{}{
		"scriptSource": "console.log('hello')",
	})

	ctx := context.Background()
	src, err := tools.GetScriptSource(ctx, client, "script1")
	if err != nil {
		t.Fatalf("GetScriptSource returned error: %v", err)
	}
	if src != "console.log('hello')" {
		t.Errorf("expected source %q, got %q", "console.log('hello')", src)
	}
}

func TestSearchInContent(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Debugger.searchInContent", map[string]interface{}{
		"result": []map[string]interface{}{
			{"lineNumber": 5, "lineContent": "hello world"},
		},
	})

	ctx := context.Background()
	result, err := tools.SearchInContent(ctx, client, "script1", "hello", true, false)
	if err != nil {
		t.Fatalf("SearchInContent returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestEvaluateOnCallFrame(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Debugger.evaluateOnCallFrame", map[string]interface{}{
		"result": map[string]interface{}{
			"type":  "number",
			"value": 42,
		},
		"wasThrown": false,
	})

	ctx := context.Background()
	result, err := tools.EvaluateOnCallFrame(ctx, client, "frame1", "x", true)
	if err != nil {
		t.Fatalf("EvaluateOnCallFrame returned error: %v", err)
	}
	if result.Result.Type != "number" {
		t.Errorf("expected result type %q, got %q", "number", result.Result.Type)
	}
	if result.WasThrown {
		t.Errorf("expected wasThrown to be false")
	}
}

func TestEvaluateOnCallFrameThrown(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Debugger.evaluateOnCallFrame", map[string]interface{}{
		"result": map[string]interface{}{
			"type": "object",
		},
		"wasThrown": true,
		"exceptionDetails": map[string]interface{}{
			"text": "ReferenceError: x is not defined",
			"exception": map[string]interface{}{
				"type":        "object",
				"description": "ReferenceError: x is not defined",
			},
		},
	})

	ctx := context.Background()
	result, err := tools.EvaluateOnCallFrame(ctx, client, "frame1", "x", true)
	if err == nil {
		t.Fatal("expected error for thrown exception, got nil")
	}
	if result == nil {
		t.Fatal("expected non-nil result even on thrown exception")
	}
	if !result.WasThrown {
		t.Errorf("expected wasThrown to be true")
	}
}

func TestSetPauseOnExceptions(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Debugger.setPauseOnExceptions", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.SetPauseOnExceptions(ctx, client, "all"); err != nil {
		t.Fatalf("SetPauseOnExceptions returned error: %v", err)
	}
}

func TestSetDOMBreakpoint(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("DOMDebugger.setDOMBreakpoint", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.SetDOMBreakpoint(ctx, client, 42, "subtree-modified"); err != nil {
		t.Fatalf("SetDOMBreakpoint returned error: %v", err)
	}
}

func TestRemoveDOMBreakpoint(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("DOMDebugger.removeDOMBreakpoint", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.RemoveDOMBreakpoint(ctx, client, 42, "subtree-modified"); err != nil {
		t.Fatalf("RemoveDOMBreakpoint returned error: %v", err)
	}
}

func TestSetEventBreakpoint(t *testing.T) {
	mock, client := setup(t)

	var receivedType, receivedEvent string
	mock.Handle("DOMDebugger.setEventBreakpoint", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			BreakpointType string `json:"breakpointType"`
			EventName      string `json:"eventName"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedType = p.BreakpointType
		receivedEvent = p.EventName
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.SetEventBreakpoint(ctx, client, "listener", "click"); err != nil {
		t.Fatalf("SetEventBreakpoint returned error: %v", err)
	}
	if receivedType != "listener" {
		t.Errorf("expected breakpointType %q, got %q", "listener", receivedType)
	}
	if receivedEvent != "click" {
		t.Errorf("expected eventName %q, got %q", "click", receivedEvent)
	}
}

func TestRemoveEventBreakpoint(t *testing.T) {
	mock, client := setup(t)

	var receivedType, receivedEvent string
	mock.Handle("DOMDebugger.removeEventBreakpoint", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			BreakpointType string `json:"breakpointType"`
			EventName      string `json:"eventName"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedType = p.BreakpointType
		receivedEvent = p.EventName
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.RemoveEventBreakpoint(ctx, client, "listener", "click"); err != nil {
		t.Fatalf("RemoveEventBreakpoint returned error: %v", err)
	}
	if receivedType != "listener" {
		t.Errorf("expected breakpointType %q, got %q", "listener", receivedType)
	}
	if receivedEvent != "click" {
		t.Errorf("expected eventName %q, got %q", "click", receivedEvent)
	}
}

func TestSetURLBreakpoint(t *testing.T) {
	mock, client := setup(t)

	var receivedURL string
	var receivedIsRegex bool
	mock.Handle("DOMDebugger.setURLBreakpoint", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			URL     string `json:"url"`
			IsRegex bool   `json:"isRegex"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedURL = p.URL
		receivedIsRegex = p.IsRegex
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.SetURLBreakpoint(ctx, client, "https://example.com/api.*", true); err != nil {
		t.Fatalf("SetURLBreakpoint returned error: %v", err)
	}
	if receivedURL != "https://example.com/api.*" {
		t.Errorf("expected url %q, got %q", "https://example.com/api.*", receivedURL)
	}
	if !receivedIsRegex {
		t.Errorf("expected isRegex to be true")
	}
}

func TestRemoveURLBreakpoint(t *testing.T) {
	mock, client := setup(t)

	var receivedURL string
	mock.Handle("DOMDebugger.removeURLBreakpoint", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedURL = p.URL
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.RemoveURLBreakpoint(ctx, client, "https://example.com/api"); err != nil {
		t.Fatalf("RemoveURLBreakpoint returned error: %v", err)
	}
	if receivedURL != "https://example.com/api" {
		t.Errorf("expected url %q, got %q", "https://example.com/api", receivedURL)
	}
}

// ---------------------------------------------------------------------------
// Timeline domain
// ---------------------------------------------------------------------------

func TestTimelineStart(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Timeline.start", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.TimelineStart(ctx, client, 0); err != nil {
		t.Fatalf("TimelineStart returned error: %v", err)
	}
}

func TestTimelineStartWithDepth(t *testing.T) {
	mock, client := setup(t)

	var receivedDepth float64
	mock.Handle("Timeline.start", func(_ string, params json.RawMessage) (interface{}, error) {
		var p map[string]interface{}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		if d, ok := p["maxCallStackDepth"]; ok {
			receivedDepth = d.(float64)
		}
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.TimelineStart(ctx, client, 10); err != nil {
		t.Fatalf("TimelineStart returned error: %v", err)
	}
	if int(receivedDepth) != 10 {
		t.Errorf("expected maxCallStackDepth 10, got %v", receivedDepth)
	}
}

func TestTimelineStop(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Timeline.stop", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.TimelineStop(ctx, client); err != nil {
		t.Fatalf("TimelineStop returned error: %v", err)
	}
}

func TestTimelineCollector(t *testing.T) {
	mock, client := setup(t)

	// Register handler for Timeline.start so collector.Start() succeeds.
	mock.HandleFunc("Timeline.start", map[string]interface{}{})

	collector := tools.NewTimelineCollector()
	ctx := context.Background()
	if err := collector.Start(ctx, client, 0); err != nil {
		t.Fatalf("TimelineCollector.Start: %v", err)
	}

	// Send a Timeline.eventRecorded event from the mock server.
	err := mock.SendEvent("Timeline.eventRecorded", map[string]interface{}{
		"record": map[string]interface{}{
			"type": "FunctionCall",
			"data": map[string]interface{}{
				"functionName": "test",
			},
			"startTime": 1000,
			"endTime":   1001,
		},
	})
	if err != nil {
		t.Fatalf("SendEvent: %v", err)
	}

	// Allow time for the event handler to process.
	time.Sleep(100 * time.Millisecond)

	events := collector.GetEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 timeline event, got %d", len(events))
	}
	if events[0].Type != "FunctionCall" {
		t.Errorf("expected event type %q, got %q", "FunctionCall", events[0].Type)
	}
	if events[0].StartTime != 1000 {
		t.Errorf("expected startTime 1000, got %v", events[0].StartTime)
	}
	if events[0].EndTime != 1001 {
		t.Errorf("expected endTime 1001, got %v", events[0].EndTime)
	}
}

// ---------------------------------------------------------------------------
// Memory domain
// ---------------------------------------------------------------------------

func TestMemoryTrackingCollector(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Memory.startTracking", map[string]interface{}{})
	mock.HandleFunc("Memory.stopTracking", map[string]interface{}{})

	collector := tools.NewMemoryTrackingCollector()
	ctx := context.Background()
	if err := collector.Start(ctx, client); err != nil {
		t.Fatalf("MemoryTrackingCollector.Start: %v", err)
	}

	// Send a trackingUpdate event.
	if err := mock.SendEvent("Memory.trackingUpdate", map[string]interface{}{
		"event": map[string]interface{}{
			"timestamp": 1000.0,
			"categories": []map[string]interface{}{
				{"type": "JavaScript", "size": 1024},
				{"type": "Images", "size": 2048},
			},
		},
	}); err != nil {
		t.Fatalf("SendEvent trackingUpdate: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Send trackingComplete to signal stop.
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = mock.SendEvent("Memory.trackingComplete", map[string]interface{}{
			"timestamp": 2000.0,
		})
	}()

	result, err := collector.Stop(ctx, client)
	if err != nil {
		t.Fatalf("MemoryTrackingCollector.Stop: %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result.Events))
	}
	if len(result.Events[0].Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(result.Events[0].Categories))
	}
	if result.Events[0].Categories[0].Type != "JavaScript" {
		t.Errorf("expected category type JavaScript, got %q", result.Events[0].Categories[0].Type)
	}
	if result.Events[0].Categories[0].Size != 1024 {
		t.Errorf("expected category size 1024, got %d", result.Events[0].Categories[0].Size)
	}
}

func TestHeapSnapshot(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Heap.snapshot", map[string]interface{}{
		"snapshotData": "snapshot-content",
	})

	ctx := context.Background()
	filePath, err := tools.HeapSnapshot(ctx, client)
	if err != nil {
		t.Fatalf("HeapSnapshot returned error: %v", err)
	}
	if filePath == "" {
		t.Fatal("expected non-empty file path")
	}
	defer func() { _ = os.Remove(filePath) }()
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read snapshot file: %v", err)
	}
	if string(data) != "snapshot-content" {
		t.Errorf("expected %q, got %q", "snapshot-content", string(data))
	}
}

func TestHeapTrackingCollector(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Heap.enable", map[string]interface{}{})
	mock.HandleFunc("Heap.gc", map[string]interface{}{})
	mock.HandleFunc("Heap.startTracking", map[string]interface{}{})
	mock.HandleFunc("Heap.stopTracking", map[string]interface{}{})

	collector := tools.NewHeapTrackingCollector()
	ctx := context.Background()

	// Send trackingStart shortly after startTracking to confirm pipeline health.
	// In production, this event carries 50-200MB+ snapshot data; here we use a
	// small mock payload. Start() waits for this signal before returning.
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = mock.SendEvent("Heap.trackingStart", map[string]interface{}{
			"timestamp":    1000.0,
			"snapshotData": "mock-snapshot",
		})
	}()

	if err := collector.Start(ctx, client); err != nil {
		t.Fatalf("HeapTrackingCollector.Start: %v", err)
	}
	if !collector.PipelineHealthy() {
		t.Error("expected pipeline to be healthy after trackingStart event")
	}

	// Send garbageCollected events.
	if err := mock.SendEvent("Heap.garbageCollected", map[string]interface{}{
		"collection": map[string]interface{}{
			"type":      "full",
			"startTime": 1100.0,
			"endTime":   1150.0,
		},
	}); err != nil {
		t.Fatalf("SendEvent garbageCollected: %v", err)
	}
	if err := mock.SendEvent("Heap.garbageCollected", map[string]interface{}{
		"collection": map[string]interface{}{
			"type":      "partial",
			"startTime": 1200.0,
			"endTime":   1210.0,
		},
	}); err != nil {
		t.Fatalf("SendEvent garbageCollected 2: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	result, err := collector.Stop(ctx, client)
	if err != nil {
		t.Fatalf("HeapTrackingCollector.Stop: %v", err)
	}
	if !result.PipelineHealthy {
		t.Error("expected PipelineHealthy=true in result")
	}
	if len(result.GCEvents) != 2 {
		t.Fatalf("expected 2 GC events, got %d", len(result.GCEvents))
	}
	if result.GCEvents[0].Type != "full" {
		t.Errorf("expected GC type %q, got %q", "full", result.GCEvents[0].Type)
	}
	if result.GCEvents[1].Type != "partial" {
		t.Errorf("expected GC type %q, got %q", "partial", result.GCEvents[1].Type)
	}
}

func TestHeapGC(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Heap.gc", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.HeapGC(ctx, client); err != nil {
		t.Fatalf("HeapGC returned error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Profiler domain
// ---------------------------------------------------------------------------

func TestCPUProfilerCollector(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("CPUProfiler.startTracking", map[string]interface{}{})
	mock.HandleFunc("CPUProfiler.stopTracking", map[string]interface{}{})

	collector := tools.NewCPUProfilerCollector()
	ctx := context.Background()
	if err := collector.Start(ctx, client); err != nil {
		t.Fatalf("CPUProfilerCollector.Start: %v", err)
	}

	// Send a trackingUpdate event.
	if err := mock.SendEvent("CPUProfiler.trackingUpdate", map[string]interface{}{
		"event": map[string]interface{}{
			"timestamp": 1000.0,
			"usage":     42.5,
		},
	}); err != nil {
		t.Fatalf("SendEvent trackingUpdate: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Send trackingComplete to signal stop.
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = mock.SendEvent("CPUProfiler.trackingComplete", map[string]interface{}{
			"timestamp": 2000.0,
		})
	}()

	result, err := collector.Stop(ctx, client)
	if err != nil {
		t.Fatalf("CPUProfilerCollector.Stop: %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Usage != 42.5 {
		t.Errorf("expected usage 42.5, got %f", result.Events[0].Usage)
	}
}

func TestScriptProfilerCollector(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("ScriptProfiler.startTracking", map[string]interface{}{})
	mock.HandleFunc("ScriptProfiler.stopTracking", map[string]interface{}{})

	collector := tools.NewScriptProfilerCollector()
	ctx := context.Background()
	if err := collector.Start(ctx, client); err != nil {
		t.Fatalf("ScriptProfilerCollector.Start: %v", err)
	}

	// Send a trackingUpdate event.
	if err := mock.SendEvent("ScriptProfiler.trackingUpdate", map[string]interface{}{
		"event": map[string]interface{}{
			"startTime": 100.0,
			"endTime":   200.0,
			"type":      "API",
		},
	}); err != nil {
		t.Fatalf("SendEvent trackingUpdate: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Send trackingComplete with samples.
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = mock.SendEvent("ScriptProfiler.trackingComplete", map[string]interface{}{
			"timestamp": 300.0,
			"samples": map[string]interface{}{
				"stackTraces": []interface{}{},
			},
		})
	}()

	result, err := collector.Stop(ctx, client)
	if err != nil {
		t.Fatalf("ScriptProfilerCollector.Stop: %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Type != "API" {
		t.Errorf("expected type API, got %q", result.Events[0].Type)
	}
	if result.Samples == nil {
		t.Error("expected non-nil samples")
	}
}

// ---------------------------------------------------------------------------
// Animation domain
// ---------------------------------------------------------------------------

func TestAnimationEnable(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Animation.enable", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.AnimationEnable(ctx, client); err != nil {
		t.Fatalf("AnimationEnable returned error: %v", err)
	}
}

func TestAnimationDisable(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Animation.disable", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.AnimationDisable(ctx, client); err != nil {
		t.Fatalf("AnimationDisable returned error: %v", err)
	}
}

func TestAnimationTrackingCollector(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Animation.startTracking", map[string]interface{}{})
	mock.HandleFunc("Animation.stopTracking", map[string]interface{}{})

	collector := tools.NewAnimationTrackingCollector()
	ctx := context.Background()
	if err := collector.Start(ctx, client); err != nil {
		t.Fatalf("AnimationTrackingCollector.Start: %v", err)
	}

	// Send a trackingUpdate event.
	if err := mock.SendEvent("Animation.trackingUpdate", map[string]interface{}{
		"timestamp": 1000.0,
		"event": map[string]interface{}{
			"trackingAnimationId": "anim-1",
			"animationState":      "active",
			"animationName":       "fadeIn",
		},
	}); err != nil {
		t.Fatalf("SendEvent trackingUpdate: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Send trackingComplete to signal stop.
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = mock.SendEvent("Animation.trackingComplete", map[string]interface{}{
			"timestamp": 2000.0,
		})
	}()

	result, err := collector.Stop(ctx, client)
	if err != nil {
		t.Fatalf("AnimationTrackingCollector.Stop: %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Event.AnimationState != "active" {
		t.Errorf("expected state active, got %q", result.Events[0].Event.AnimationState)
	}
	if result.Events[0].Event.AnimationName != "fadeIn" {
		t.Errorf("expected animation name fadeIn, got %q", result.Events[0].Event.AnimationName)
	}
}

func TestGetAnimationEffect(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Animation.requestEffectTarget", map[string]interface{}{
		"nodeId": 99,
	})

	ctx := context.Background()
	result, err := tools.GetAnimationEffect(ctx, client, "anim-1")
	if err != nil {
		t.Fatalf("GetAnimationEffect returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}
	if int(parsed["nodeId"].(float64)) != 99 {
		t.Errorf("expected nodeId 99, got %v", parsed["nodeId"])
	}
}

func TestResolveAnimation(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Animation.resolveAnimation", map[string]interface{}{
		"object": map[string]interface{}{
			"type":     "object",
			"objectId": "anim-obj-1",
		},
	})

	ctx := context.Background()
	obj, err := tools.ResolveAnimation(ctx, client, "anim-1", "test-group")
	if err != nil {
		t.Fatalf("ResolveAnimation returned error: %v", err)
	}
	if obj == nil {
		t.Fatal("expected non-nil object")
	}
	if obj.Type != "object" {
		t.Errorf("expected object type %q, got %q", "object", obj.Type)
	}
	if obj.ObjectID != "anim-obj-1" {
		t.Errorf("expected objectId %q, got %q", "anim-obj-1", obj.ObjectID)
	}
}

// ---------------------------------------------------------------------------
// Canvas domain
// ---------------------------------------------------------------------------

func TestCanvasEnable(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Canvas.enable", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.CanvasEnable(ctx, client); err != nil {
		t.Fatalf("CanvasEnable returned error: %v", err)
	}
}

func TestCanvasDisable(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Canvas.disable", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.CanvasDisable(ctx, client); err != nil {
		t.Fatalf("CanvasDisable returned error: %v", err)
	}
}

func TestGetCanvasContent(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Canvas.requestContent", map[string]interface{}{
		"content": "data:image/png;base64,abc",
	})

	ctx := context.Background()
	content, err := tools.GetCanvasContent(ctx, client, "canvas-1")
	if err != nil {
		t.Fatalf("GetCanvasContent returned error: %v", err)
	}
	if content != "data:image/png;base64,abc" {
		t.Errorf("expected content %q, got %q", "data:image/png;base64,abc", content)
	}
}

func TestStartCanvasRecording(t *testing.T) {
	mock, client := setup(t)

	var receivedFrameCount float64
	mock.Handle("Canvas.startRecording", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			CanvasID   string  `json:"canvasId"`
			FrameCount float64 `json:"frameCount"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedFrameCount = p.FrameCount
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.StartCanvasRecording(ctx, client, "canvas-1", 1); err != nil {
		t.Fatalf("StartCanvasRecording returned error: %v", err)
	}
	if int(receivedFrameCount) != 1 {
		t.Errorf("expected frameCount 1, got %v", receivedFrameCount)
	}
}

func TestStopCanvasRecording(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Canvas.stopRecording", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.StopCanvasRecording(ctx, client, "canvas-1"); err != nil {
		t.Fatalf("StopCanvasRecording returned error: %v", err)
	}
}

func TestGetShaderSource(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Canvas.requestShaderSource", map[string]interface{}{
		"content": "void main() {}",
	})

	ctx := context.Background()
	src, err := tools.GetShaderSource(ctx, client, "prog-1", "vertex")
	if err != nil {
		t.Fatalf("GetShaderSource returned error: %v", err)
	}
	if src != "void main() {}" {
		t.Errorf("expected shader source %q, got %q", "void main() {}", src)
	}
}

// ---------------------------------------------------------------------------
// LayerTree domain
// ---------------------------------------------------------------------------

func TestGetLayerTree(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("LayerTree.layersForNode", map[string]interface{}{
		"layers": []map[string]interface{}{
			{"layerId": "layer-1", "nodeId": 1, "bounds": map[string]interface{}{"x": 0, "y": 0, "width": 100, "height": 100}},
		},
	})

	ctx := context.Background()
	result, err := tools.GetLayerTree(ctx, client, 1)
	if err != nil {
		t.Fatalf("GetLayerTree returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}
	layers, ok := parsed["layers"].([]interface{})
	if !ok {
		t.Fatal("expected layers array in result")
	}
	if len(layers) != 1 {
		t.Errorf("expected 1 layer, got %d", len(layers))
	}
}

func TestGetCompositingReasons(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("LayerTree.reasonsForCompositingLayer", map[string]interface{}{
		"compositingReasons": []string{"transform", "overlap"},
	})

	ctx := context.Background()
	result, err := tools.GetCompositingReasons(ctx, client, "layer-1")
	if err != nil {
		t.Fatalf("GetCompositingReasons returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}
	reasons, ok := parsed["compositingReasons"].([]interface{})
	if !ok {
		t.Fatal("expected compositingReasons array in result")
	}
	if len(reasons) != 2 {
		t.Errorf("expected 2 compositing reasons, got %d", len(reasons))
	}
}

// ---------------------------------------------------------------------------
// Worker domain
// ---------------------------------------------------------------------------

func TestWorkerEnable(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Worker.enable", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.WorkerEnable(ctx, client); err != nil {
		t.Fatalf("WorkerEnable returned error: %v", err)
	}
}

func TestWorkerDisable(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Worker.disable", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.WorkerDisable(ctx, client); err != nil {
		t.Fatalf("WorkerDisable returned error: %v", err)
	}
}

func TestSendToWorker(t *testing.T) {
	mock, client := setup(t)

	var receivedWorkerID, receivedMessage string
	mock.Handle("Worker.sendMessageToWorker", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			WorkerID string `json:"workerId"`
			Message  string `json:"message"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedWorkerID = p.WorkerID
		receivedMessage = p.Message
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.SendToWorker(ctx, client, "worker-1", `{"method":"echo"}`); err != nil {
		t.Fatalf("SendToWorker returned error: %v", err)
	}
	if receivedWorkerID != "worker-1" {
		t.Errorf("expected workerId %q, got %q", "worker-1", receivedWorkerID)
	}
	if receivedMessage != `{"method":"echo"}` {
		t.Errorf("expected message %q, got %q", `{"method":"echo"}`, receivedMessage)
	}
}

func TestGetServiceWorkerInfo(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Runtime.evaluate", map[string]interface{}{
		"result": map[string]interface{}{
			"type": "object",
			"value": map[string]interface{}{
				"supported":     true,
				"controller":    nil,
				"registrations": []interface{}{},
			},
		},
		"wasThrown": false,
	})

	ctx := context.Background()
	result, err := tools.GetServiceWorkerInfo(ctx, client)
	if err != nil {
		t.Fatalf("GetServiceWorkerInfo returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// ---------------------------------------------------------------------------
// Audit domain
// ---------------------------------------------------------------------------

func TestRunAudit(t *testing.T) {
	mock, client := setup(t)

	mock.HandleFunc("Audit.setup", map[string]interface{}{})
	mock.HandleFunc("Audit.run", map[string]interface{}{
		"result": map[string]interface{}{
			"passed": true,
			"errors": []interface{}{},
		},
	})
	mock.HandleFunc("Audit.teardown", map[string]interface{}{})

	ctx := context.Background()
	result, err := tools.RunAudit(ctx, client, "testCode")
	if err != nil {
		t.Fatalf("RunAudit returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}
	r, ok := parsed["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result object in response")
	}
	if r["passed"] != true {
		t.Errorf("expected passed to be true, got %v", r["passed"])
	}
}

// ---------------------------------------------------------------------------
// Security domain
// ---------------------------------------------------------------------------

func TestGetCertificateInfo(t *testing.T) {
	mock, client := setup(t)
	mock.HandleFunc("Network.getSerializedCertificate", map[string]interface{}{
		"serializedCertificate": "MIIB...",
	})

	ctx := context.Background()
	result, err := tools.GetCertificateInfo(ctx, client, "req-1")
	if err != nil {
		t.Fatalf("GetCertificateInfo returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}
	if parsed["serializedCertificate"] != "MIIB..." {
		t.Errorf("expected serializedCertificate %q, got %v", "MIIB...", parsed["serializedCertificate"])
	}
}
