package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// GetMatchedStyles returns the matched CSS rules for a node.
func GetMatchedStyles(ctx context.Context, client *webkit.Client, nodeID int) (json.RawMessage, error) {
	result, err := client.Send(ctx, "CSS.getMatchedStylesForNode", map[string]interface{}{
		"nodeId": nodeID,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetComputedStyle returns the computed style properties for a node.
func GetComputedStyle(ctx context.Context, client *webkit.Client, nodeID int) ([]webkit.CSSProperty, error) {
	result, err := client.Send(ctx, "CSS.getComputedStyleForNode", map[string]interface{}{
		"nodeId": nodeID,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		ComputedStyle []webkit.CSSProperty `json:"computedStyle"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("decoding computed style: %w", err)
	}
	return resp.ComputedStyle, nil
}

// GetInlineStyles returns the inline style for a node.
func GetInlineStyles(ctx context.Context, client *webkit.Client, nodeID int) (*webkit.CSSStyle, error) {
	result, err := client.Send(ctx, "CSS.getInlineStylesForNode", map[string]interface{}{
		"nodeId": nodeID,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		InlineStyle *webkit.CSSStyle `json:"inlineStyle"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("decoding inline styles: %w", err)
	}
	return resp.InlineStyle, nil
}

// SetStyleText edits a style's text content.
func SetStyleText(ctx context.Context, client *webkit.Client, styleID json.RawMessage, text string) (*webkit.CSSStyle, error) {
	result, err := client.Send(ctx, "CSS.setStyleText", map[string]interface{}{
		"styleId": styleID,
		"text":    text,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Style *webkit.CSSStyle `json:"style"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("decoding set style result: %w", err)
	}
	return resp.Style, nil
}

// errCSSNotSupported is returned when CSS.enable/CSS.getAllStyleSheets/CSS.getStyleSheetText
// are called through iwdp Target routing, which hangs and corrupts the connection pipeline.
var errCSSNotSupported = fmt.Errorf("CSS.enable/CSS.getAllStyleSheets/CSS.getStyleSheetText do not work through ios-webkit-debug-proxy Target routing (the commands hang without a response, breaking subsequent commands on the same connection)")

// GetAllStylesheets returns all stylesheets known to the page.
// Note: this requires CSS.enable which does not work through iwdp Target routing.
// When connected via iwdp, returns an error immediately to avoid hanging the connection.
func GetAllStylesheets(ctx context.Context, client *webkit.Client) ([]webkit.CSSStyleSheet, error) {
	if client.IsTargetRouted() {
		return nil, errCSSNotSupported
	}

	result, err := client.Send(ctx, "CSS.getAllStyleSheets", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Headers []webkit.CSSStyleSheet `json:"headers"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("decoding stylesheets: %w", err)
	}
	return resp.Headers, nil
}

// GetStylesheetText returns the text content of a stylesheet.
// Note: this requires CSS.enable which does not work through iwdp Target routing.
// When connected via iwdp, returns an error immediately to avoid hanging the connection.
func GetStylesheetText(ctx context.Context, client *webkit.Client, styleSheetID string) (string, error) {
	if client.IsTargetRouted() {
		return "", errCSSNotSupported
	}

	result, err := client.Send(ctx, "CSS.getStyleSheetText", map[string]interface{}{
		"styleSheetId": styleSheetID,
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", fmt.Errorf("decoding stylesheet text: %w", err)
	}
	return resp.Text, nil
}

// ForcePseudoState forces the given CSS pseudo-classes on a node.
func ForcePseudoState(ctx context.Context, client *webkit.Client, nodeID int, pseudoClasses []string) error {
	_, err := client.Send(ctx, "CSS.forcePseudoState", map[string]interface{}{
		"nodeId":              nodeID,
		"forcedPseudoClasses": pseudoClasses,
	})
	return err
}
