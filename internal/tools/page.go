package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// Navigate navigates the page to the given URL.
func Navigate(ctx context.Context, client *webkit.Client, url string) error {
	_, err := client.Send(ctx, "Page.navigate", map[string]string{"url": url})
	return err
}

// Reload reloads the current page.
func Reload(ctx context.Context, client *webkit.Client, ignoreCache bool) error {
	_, err := client.Send(ctx, "Page.reload", map[string]interface{}{
		"ignoreCache": ignoreCache,
	})
	return err
}

// TakeScreenshot captures the page as a base64-encoded PNG.
func TakeScreenshot(ctx context.Context, client *webkit.Client) (string, error) {
	// Get page dimensions — Page.snapshotRect requires explicit width/height.
	dimResult, err := client.Send(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression": `JSON.stringify({
			width: Math.max(
				document.body.scrollWidth, document.documentElement.scrollWidth,
				document.body.offsetWidth, document.documentElement.offsetWidth,
				document.body.clientWidth, document.documentElement.clientWidth
			),
			height: Math.max(
				document.body.scrollHeight, document.documentElement.scrollHeight,
				document.body.offsetHeight, document.documentElement.offsetHeight,
				document.body.clientHeight, document.documentElement.clientHeight
			)
		})`,
		"returnByValue": true,
	})
	if err != nil {
		return "", fmt.Errorf("getting page dimensions: %w", err)
	}

	var evalResp struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(dimResult, &evalResp); err != nil {
		return "", fmt.Errorf("decoding dimension result: %w", err)
	}

	var dims struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	dimStr := string(evalResp.Result.Value)
	// Value comes back as a JSON string (since we used JSON.stringify), so unquote first.
	var unquoted string
	if err := json.Unmarshal(evalResp.Result.Value, &unquoted); err == nil {
		dimStr = unquoted
	}
	if err := json.Unmarshal([]byte(dimStr), &dims); err != nil {
		return "", fmt.Errorf("parsing page dimensions: %w", err)
	}

	if dims.Width <= 0 || dims.Height <= 0 {
		return "", fmt.Errorf("invalid page dimensions: %dx%d", dims.Width, dims.Height)
	}

	result, err := client.Send(ctx, "Page.snapshotRect", map[string]interface{}{
		"x":                0,
		"y":                0,
		"width":            dims.Width,
		"height":           dims.Height,
		"coordinateSystem": "Page",
	})
	if err != nil {
		return "", err
	}

	var snap struct {
		DataURL string `json:"dataURL"`
	}
	if err := json.Unmarshal(result, &snap); err != nil {
		return "", fmt.Errorf("decoding screenshot: %w", err)
	}
	return snap.DataURL, nil
}

// SnapshotNode captures a specific DOM node as a base64-encoded PNG.
func SnapshotNode(ctx context.Context, client *webkit.Client, nodeID int) (string, error) {
	result, err := client.Send(ctx, "Page.snapshotNode", map[string]interface{}{
		"nodeId": nodeID,
	})
	if err != nil {
		return "", err
	}

	var snap struct {
		DataURL string `json:"dataURL"`
	}
	if err := json.Unmarshal(result, &snap); err != nil {
		return "", fmt.Errorf("decoding node snapshot: %w", err)
	}
	return snap.DataURL, nil
}

// GetCookies returns all cookies for the current page, including httpOnly and secure.
func GetCookies(ctx context.Context, client *webkit.Client) ([]webkit.Cookie, error) {
	result, err := client.Send(ctx, "Page.getCookies", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Cookies []webkit.Cookie `json:"cookies"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("decoding cookies: %w", err)
	}
	return resp.Cookies, nil
}

// SetCookie sets a cookie.
func SetCookie(ctx context.Context, client *webkit.Client, cookie webkit.Cookie) error {
	_, err := client.Send(ctx, "Page.setCookie", map[string]interface{}{
		"cookie": cookie,
	})
	return err
}

// DeleteCookie deletes a cookie by name and URL.
func DeleteCookie(ctx context.Context, client *webkit.Client, name, url string) error {
	_, err := client.Send(ctx, "Page.deleteCookie", map[string]string{
		"cookieName": name,
		"url":        url,
	})
	return err
}
