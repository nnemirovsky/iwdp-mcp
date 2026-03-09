package tools

import (
	"context"
	"encoding/json"
	"fmt"

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

// GetServiceWorkerInfo retrieves service worker registrations for the current page
// via the navigator.serviceWorker JS API (works on page connections without the
// ServiceWorker inspector domain).
func GetServiceWorkerInfo(ctx context.Context, client *webkit.Client) (json.RawMessage, error) {
	const script = `(async () => {
		if (!navigator.serviceWorker) return {supported: false};
		const regs = await navigator.serviceWorker.getRegistrations();
		const controller = navigator.serviceWorker.controller;
		return {
			supported: true,
			controller: controller ? {
				scriptURL: controller.scriptURL,
				state: controller.state,
			} : null,
			registrations: regs.map(r => ({
				scope: r.scope,
				active: r.active ? {scriptURL: r.active.scriptURL, state: r.active.state} : null,
				waiting: r.waiting ? {scriptURL: r.waiting.scriptURL, state: r.waiting.state} : null,
				installing: r.installing ? {scriptURL: r.installing.scriptURL, state: r.installing.state} : null,
			})),
		};
	})()`
	result, err := client.Send(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression":    script,
		"returnByValue": true,
		"awaitPromise":  true,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		WasThrown bool `json:"wasThrown"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, err
	}
	if resp.WasThrown {
		return nil, fmt.Errorf("service worker query failed")
	}
	if len(resp.Result.Value) == 0 {
		return json.RawMessage(`{"supported": false}`), nil
	}
	return resp.Result.Value, nil
}
