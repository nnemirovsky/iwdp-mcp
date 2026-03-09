package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// GetDocument returns the root DOM node.
func GetDocument(ctx context.Context, client *webkit.Client, depth int) (*webkit.Node, error) {
	params := map[string]interface{}{}
	if depth > 0 {
		params["depth"] = depth
	}

	result, err := client.Send(ctx, "DOM.getDocument", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Root *webkit.Node `json:"root"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("decoding document: %w", err)
	}
	return resp.Root, nil
}

// QuerySelector finds the first element matching a CSS selector.
func QuerySelector(ctx context.Context, client *webkit.Client, nodeID int, selector string) (int, error) {
	result, err := client.Send(ctx, "DOM.querySelector", map[string]interface{}{
		"nodeId":   nodeID,
		"selector": selector,
	})
	if err != nil {
		return 0, err
	}

	var resp struct {
		NodeID int `json:"nodeId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return 0, fmt.Errorf("decoding querySelector: %w", err)
	}
	if resp.NodeID == 0 {
		return 0, fmt.Errorf("no element matches selector %q", selector)
	}
	return resp.NodeID, nil
}

// QuerySelectorAll finds all elements matching a CSS selector.
func QuerySelectorAll(ctx context.Context, client *webkit.Client, nodeID int, selector string) ([]int, error) {
	result, err := client.Send(ctx, "DOM.querySelectorAll", map[string]interface{}{
		"nodeId":   nodeID,
		"selector": selector,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		NodeIDs []int `json:"nodeIds"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("decoding querySelectorAll: %w", err)
	}
	return resp.NodeIDs, nil
}

// GetOuterHTML returns the outer HTML for a node.
func GetOuterHTML(ctx context.Context, client *webkit.Client, nodeID int) (string, error) {
	result, err := client.Send(ctx, "DOM.getOuterHTML", map[string]interface{}{
		"nodeId": nodeID,
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		OuterHTML string `json:"outerHTML"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", fmt.Errorf("decoding outerHTML: %w", err)
	}
	return resp.OuterHTML, nil
}

// GetAttributes returns attributes of a node as key-value pairs.
func GetAttributes(ctx context.Context, client *webkit.Client, nodeID int) (map[string]string, error) {
	result, err := client.Send(ctx, "DOM.getAttributes", map[string]interface{}{
		"nodeId": nodeID,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Attributes []string `json:"attributes"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("decoding attributes: %w", err)
	}

	attrs := make(map[string]string)
	for i := 0; i+1 < len(resp.Attributes); i += 2 {
		attrs[resp.Attributes[i]] = resp.Attributes[i+1]
	}
	return attrs, nil
}

// EventListenerInfo describes an event listener on a node.
type EventListenerInfo struct {
	Type        string `json:"type"`
	UseCapture  bool   `json:"useCapture"`
	IsAttribute bool   `json:"isAttribute"`
	NodeID      int    `json:"nodeId,omitempty"`
	HandlerBody string `json:"handlerBody,omitempty"`
}

// GetEventListeners returns event listeners registered on a node.
func GetEventListeners(ctx context.Context, client *webkit.Client, nodeID int) ([]EventListenerInfo, error) {
	result, err := client.Send(ctx, "DOM.getEventListenersForNode", map[string]interface{}{
		"nodeId": nodeID,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Listeners []EventListenerInfo `json:"listeners"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("decoding event listeners: %w", err)
	}
	return resp.Listeners, nil
}

// HighlightNode highlights a node in the browser.
func HighlightNode(ctx context.Context, client *webkit.Client, nodeID int) error {
	_, err := client.Send(ctx, "DOM.highlightNode", map[string]interface{}{
		"nodeId": nodeID,
		"highlightConfig": map[string]interface{}{
			"showInfo":     true,
			"contentColor": map[string]int{"r": 111, "g": 168, "b": 220, "a": 66},
			"paddingColor": map[string]int{"r": 147, "g": 196, "b": 125, "a": 55},
			"borderColor":  map[string]int{"r": 255, "g": 229, "b": 153, "a": 66},
			"marginColor":  map[string]int{"r": 246, "g": 178, "b": 107, "a": 66},
		},
	})
	return err
}

// HideHighlight removes the current DOM highlight.
func HideHighlight(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "DOM.hideHighlight", nil)
	return err
}
