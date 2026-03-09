package tools

import (
	"context"
	"encoding/json"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// RunAudit sets up the audit environment, runs the specified test, tears down,
// and returns the run result.
func RunAudit(ctx context.Context, client *webkit.Client, testStr string) (json.RawMessage, error) {
	_, err := client.Send(ctx, "Audit.setup", nil)
	if err != nil {
		return nil, err
	}

	result, err := client.Send(ctx, "Audit.run", map[string]interface{}{
		"test": testStr,
	})
	if err != nil {
		// Attempt teardown even on failure.
		_, _ = client.Send(ctx, "Audit.teardown", nil)
		return nil, err
	}

	_, _ = client.Send(ctx, "Audit.teardown", nil)
	return result, nil
}
