package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// EvaluateResult holds the result of a JavaScript evaluation.
type EvaluateResult struct {
	Result           webkit.RemoteObject      `json:"result"`
	ExceptionDetails *webkit.ExceptionDetails `json:"exceptionDetails,omitempty"`
	WasThrown        bool                     `json:"wasThrown"`
}

// EvaluateScript evaluates a JavaScript expression in the page context.
func EvaluateScript(ctx context.Context, client *webkit.Client, expression string, returnByValue bool) (*EvaluateResult, error) {
	result, err := client.Send(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression":            expression,
		"returnByValue":         returnByValue,
		"generatePreview":       true,
		"includeCommandLineAPI": false,
	})
	if err != nil {
		return nil, err
	}

	var evalResult EvaluateResult
	if err := json.Unmarshal(result, &evalResult); err != nil {
		return nil, fmt.Errorf("decoding eval result: %w", err)
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

// CallFunctionOn calls a function on a remote object.
func CallFunctionOn(ctx context.Context, client *webkit.Client, objectID, functionDeclaration string, args []interface{}, returnByValue bool) (*EvaluateResult, error) {
	params := map[string]interface{}{
		"objectId":            objectID,
		"functionDeclaration": functionDeclaration,
		"returnByValue":       returnByValue,
		"generatePreview":     true,
	}
	if len(args) > 0 {
		callArgs := make([]map[string]interface{}, len(args))
		for i, arg := range args {
			callArgs[i] = map[string]interface{}{"value": arg}
		}
		params["arguments"] = callArgs
	}

	result, err := client.Send(ctx, "Runtime.callFunctionOn", params)
	if err != nil {
		return nil, err
	}

	var evalResult EvaluateResult
	if err := json.Unmarshal(result, &evalResult); err != nil {
		return nil, fmt.Errorf("decoding call result: %w", err)
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

// GetProperties retrieves properties of a remote object.
func GetProperties(ctx context.Context, client *webkit.Client, objectID string, ownProperties bool) ([]webkit.PropertyDescriptor, error) {
	result, err := client.Send(ctx, "Runtime.getProperties", map[string]interface{}{
		"objectId":      objectID,
		"ownProperties": ownProperties,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Properties []webkit.PropertyDescriptor `json:"properties"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("decoding properties: %w", err)
	}
	return resp.Properties, nil
}
