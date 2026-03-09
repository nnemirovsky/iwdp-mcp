package tools

import (
	"context"
	"encoding/json"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// GetCertificateInfo retrieves the serialized certificate for a given network request.
func GetCertificateInfo(ctx context.Context, client *webkit.Client, requestID string) (json.RawMessage, error) {
	result, err := client.Send(ctx, "Network.getSerializedCertificate", map[string]interface{}{
		"requestId": requestID,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
