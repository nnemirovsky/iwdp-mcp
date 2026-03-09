package tools

import (
	"context"
	"encoding/json"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// WorkerEnable enables the Worker domain.
func WorkerEnable(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Worker.enable", nil)
	return err
}

// WorkerDisable disables the Worker domain.
func WorkerDisable(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Worker.disable", nil)
	return err
}

// SendToWorker sends a message to a specific worker.
func SendToWorker(ctx context.Context, client *webkit.Client, workerID, message string) error {
	_, err := client.Send(ctx, "Worker.sendMessageToWorker", map[string]interface{}{
		"workerId": workerID,
		"message":  message,
	})
	return err
}

// GetServiceWorkerInfo retrieves service worker initialization info.
func GetServiceWorkerInfo(ctx context.Context, client *webkit.Client) (json.RawMessage, error) {
	result, err := client.Send(ctx, "ServiceWorker.getInitializationInfo", nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}
