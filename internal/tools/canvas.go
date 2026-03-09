package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// CanvasEnable enables the Canvas domain.
func CanvasEnable(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Canvas.enable", nil)
	return err
}

// CanvasDisable disables the Canvas domain.
func CanvasDisable(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Canvas.disable", nil)
	return err
}

// GetCanvasContent requests the content of a canvas and returns it as a base64 data URL.
func GetCanvasContent(ctx context.Context, client *webkit.Client, canvasID string) (string, error) {
	result, err := client.Send(ctx, "Canvas.requestContent", map[string]interface{}{
		"canvasId": canvasID,
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", fmt.Errorf("decoding canvas content: %w", err)
	}
	return resp.Content, nil
}

// StartCanvasRecording starts recording actions on a canvas.
// frameCount specifies the number of frames to record (0 means unlimited).
func StartCanvasRecording(ctx context.Context, client *webkit.Client, canvasID string, frameCount int) error {
	params := map[string]interface{}{
		"canvasId": canvasID,
	}
	if frameCount > 0 {
		params["frameCount"] = frameCount
	}
	_, err := client.Send(ctx, "Canvas.startRecording", params)
	return err
}

// StopCanvasRecording stops recording actions on a canvas.
func StopCanvasRecording(ctx context.Context, client *webkit.Client, canvasID string) error {
	_, err := client.Send(ctx, "Canvas.stopRecording", map[string]interface{}{
		"canvasId": canvasID,
	})
	return err
}

// GetShaderSource retrieves the source code of a shader program.
func GetShaderSource(ctx context.Context, client *webkit.Client, programID, shaderType string) (string, error) {
	result, err := client.Send(ctx, "Canvas.requestShaderSource", map[string]interface{}{
		"programId":  programID,
		"shaderType": shaderType,
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", fmt.Errorf("decoding shader source: %w", err)
	}
	return resp.Content, nil
}
