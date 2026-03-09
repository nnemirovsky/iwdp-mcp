package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// DebuggerEnable enables the debugger for the page.
func DebuggerEnable(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Debugger.enable", nil)
	return err
}

// DebuggerDisable disables the debugger for the page.
func DebuggerDisable(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Debugger.disable", nil)
	return err
}

// SetBreakpointByURL sets a breakpoint at a given URL, line, and optional column/condition.
// If columnNumber is non-nil, it is included in the request (even if 0).
func SetBreakpointByURL(ctx context.Context, client *webkit.Client, url string, lineNumber int, columnNumber *int, condition string) (webkit.BreakpointID, []webkit.Location, error) {
	params := map[string]interface{}{
		"url":        url,
		"lineNumber": lineNumber,
	}
	if columnNumber != nil {
		params["columnNumber"] = *columnNumber
	}
	if condition != "" {
		params["condition"] = condition
	}

	result, err := client.Send(ctx, "Debugger.setBreakpointByUrl", params)
	if err != nil {
		return "", nil, err
	}

	var resp struct {
		BreakpointID webkit.BreakpointID `json:"breakpointId"`
		Locations    []webkit.Location   `json:"locations"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", nil, fmt.Errorf("decoding setBreakpointByUrl: %w", err)
	}
	return resp.BreakpointID, resp.Locations, nil
}

// RemoveBreakpoint removes a previously set breakpoint.
func RemoveBreakpoint(ctx context.Context, client *webkit.Client, breakpointID webkit.BreakpointID) error {
	_, err := client.Send(ctx, "Debugger.removeBreakpoint", map[string]interface{}{
		"breakpointId": breakpointID,
	})
	return err
}

// Pause pauses JavaScript execution.
func Pause(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Debugger.pause", nil)
	return err
}

// Resume resumes JavaScript execution.
func Resume(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Debugger.resume", nil)
	return err
}

// StepOver steps over the next statement.
func StepOver(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Debugger.stepOver", nil)
	return err
}

// StepInto steps into the next function call.
func StepInto(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Debugger.stepInto", nil)
	return err
}

// StepOut steps out of the current function.
func StepOut(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Debugger.stepOut", nil)
	return err
}

// GetScriptSource returns the source code of a script.
func GetScriptSource(ctx context.Context, client *webkit.Client, scriptID string) (string, error) {
	result, err := client.Send(ctx, "Debugger.getScriptSource", map[string]interface{}{
		"scriptId": scriptID,
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		ScriptSource string `json:"scriptSource"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", fmt.Errorf("decoding scriptSource: %w", err)
	}
	return resp.ScriptSource, nil
}

// SearchInContent searches for a query string in a script's source.
func SearchInContent(ctx context.Context, client *webkit.Client, scriptID, query string, caseSensitive, isRegex bool) (json.RawMessage, error) {
	result, err := client.Send(ctx, "Debugger.searchInContent", map[string]interface{}{
		"scriptId":      scriptID,
		"query":         query,
		"caseSensitive": caseSensitive,
		"isRegex":       isRegex,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// EvaluateOnCallFrame evaluates an expression on a given call frame.
func EvaluateOnCallFrame(ctx context.Context, client *webkit.Client, callFrameID, expression string, returnByValue bool) (*EvaluateResult, error) {
	result, err := client.Send(ctx, "Debugger.evaluateOnCallFrame", map[string]interface{}{
		"callFrameId":     callFrameID,
		"expression":      expression,
		"returnByValue":   returnByValue,
		"generatePreview": true,
	})
	if err != nil {
		return nil, err
	}

	var evalResult EvaluateResult
	if err := json.Unmarshal(result, &evalResult); err != nil {
		return nil, fmt.Errorf("decoding evaluateOnCallFrame result: %w", err)
	}

	if evalResult.WasThrown {
		text := "JavaScript error"
		if evalResult.ExceptionDetails != nil {
			text = evalResult.ExceptionDetails.Text
			if evalResult.ExceptionDetails.Exception != nil && evalResult.ExceptionDetails.Exception.Description != "" {
				text = evalResult.ExceptionDetails.Exception.Description
			}
		}
		return &evalResult, fmt.Errorf("%s", text)
	}

	return &evalResult, nil
}

// SetPauseOnExceptions configures when the debugger should pause on exceptions.
// Valid state values: "none", "uncaught", "all".
func SetPauseOnExceptions(ctx context.Context, client *webkit.Client, state string) error {
	_, err := client.Send(ctx, "Debugger.setPauseOnExceptions", map[string]interface{}{
		"state": state,
	})
	return err
}

// SetDOMBreakpoint sets a breakpoint on a DOM node.
func SetDOMBreakpoint(ctx context.Context, client *webkit.Client, nodeID int, breakpointType string) error {
	_, err := client.Send(ctx, "DOMDebugger.setDOMBreakpoint", map[string]interface{}{
		"nodeId": nodeID,
		"type":   breakpointType,
	})
	return err
}

// RemoveDOMBreakpoint removes a breakpoint from a DOM node.
func RemoveDOMBreakpoint(ctx context.Context, client *webkit.Client, nodeID int, breakpointType string) error {
	_, err := client.Send(ctx, "DOMDebugger.removeDOMBreakpoint", map[string]interface{}{
		"nodeId": nodeID,
		"type":   breakpointType,
	})
	return err
}

// SetEventBreakpoint sets a breakpoint on a named event.
func SetEventBreakpoint(ctx context.Context, client *webkit.Client, eventName string) error {
	_, err := client.Send(ctx, "DOMDebugger.setEventBreakpoint", map[string]interface{}{
		"eventName": eventName,
	})
	return err
}

// RemoveEventBreakpoint removes a breakpoint from a named event.
func RemoveEventBreakpoint(ctx context.Context, client *webkit.Client, eventName string) error {
	_, err := client.Send(ctx, "DOMDebugger.removeEventBreakpoint", map[string]interface{}{
		"eventName": eventName,
	})
	return err
}

// SetURLBreakpoint sets a breakpoint on network requests matching a URL.
func SetURLBreakpoint(ctx context.Context, client *webkit.Client, url string, isRegex bool) error {
	_, err := client.Send(ctx, "DOMDebugger.setURLBreakpoint", map[string]interface{}{
		"url":     url,
		"isRegex": isRegex,
	})
	return err
}

// RemoveURLBreakpoint removes a URL breakpoint.
func RemoveURLBreakpoint(ctx context.Context, client *webkit.Client, url string) error {
	_, err := client.Send(ctx, "DOMDebugger.removeURLBreakpoint", map[string]interface{}{
		"url": url,
	})
	return err
}
