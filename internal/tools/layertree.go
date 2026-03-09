package tools

import (
	"context"
	"encoding/json"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// GetLayerTree returns the layer tree for a given DOM node.
func GetLayerTree(ctx context.Context, client *webkit.Client, nodeID int) (json.RawMessage, error) {
	result, err := client.Send(ctx, "LayerTree.layersForNode", map[string]interface{}{
		"nodeId": nodeID,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetCompositingReasons returns the compositing reasons for a given layer.
func GetCompositingReasons(ctx context.Context, client *webkit.Client, layerID string) (json.RawMessage, error) {
	result, err := client.Send(ctx, "LayerTree.reasonsForCompositingLayer", map[string]interface{}{
		"layerId": layerID,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
