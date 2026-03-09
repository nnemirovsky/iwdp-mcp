package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"time"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

const (
	// DefaultListingPort is the port where iwdp lists all connected devices.
	// http://localhost:9221 → JSON array of devices
	DefaultListingPort = 9221

	// DefaultFirstDevicePort is the port iwdp assigns to the first device.
	// Each additional device gets the next port (9223, 9224, ...).
	// http://localhost:9222/json → JSON array of pages for that device
	DefaultFirstDevicePort = 9222
)

// IsRunning checks if ios_webkit_debug_proxy is reachable by hitting the
// listing port (9221). This is the canonical way to detect a running proxy.
func IsRunning() bool {
	u := fmt.Sprintf("http://localhost:%d/json", DefaultListingPort)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Start launches ios_webkit_debug_proxy as a background process.
// Uses a detached context so iwdp outlives the MCP server if needed.
func Start(_ context.Context) error {
	cmd := exec.Command("ios_webkit_debug_proxy", "-F")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting ios_webkit_debug_proxy: %w\nMake sure it is installed: brew install ios-webkit-debug-proxy", err)
	}
	// Don't wait — let it run in the background.
	go func() {
		_ = cmd.Wait()
	}()

	// Wait briefly and verify it started via the listing port.
	for i := 0; i < 10; i++ {
		time.Sleep(300 * time.Millisecond)
		if IsRunning() {
			return nil
		}
	}
	return fmt.Errorf("ios_webkit_debug_proxy started but not responding on port %d", DefaultListingPort)
}

// EnsureRunning checks if iwdp is running and starts it if not.
// If iwdp crashed (e.g., after a large heap snapshot), this will restart it.
func EnsureRunning(ctx context.Context) error {
	if IsRunning() {
		return nil
	}
	return Start(ctx)
}

// Restart kills any existing iwdp process and starts a fresh one.
// Use this when iwdp is in a bad state (e.g., after a WebSocket crash).
func Restart(ctx context.Context) error {
	// Best-effort kill — use exact binary name to avoid matching unrelated processes
	_ = exec.Command("pkill", "-x", "ios_webkit_debug_proxy").Run()
	// Wait for the process to fully exit and release the port
	for i := 0; i < 10; i++ {
		time.Sleep(300 * time.Millisecond)
		if !IsRunning() {
			break
		}
	}
	return Start(ctx)
}

// ListDevices fetches connected devices from the listing port (9221).
// Each device entry contains a URL like "localhost:9222" indicating
// which port to query for that device's pages.
func ListDevices() ([]webkit.DeviceEntry, error) {
	return listDevicesFromURL(fmt.Sprintf("http://localhost:%d/json", DefaultListingPort))
}

func listDevicesFromURL(u string) ([]webkit.DeviceEntry, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("connecting to iwdp: %w\nIs ios_webkit_debug_proxy running?", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("iwdp returned HTTP %d for %s", resp.StatusCode, u)
	}

	var devices []webkit.DeviceEntry
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		return nil, fmt.Errorf("decoding device list: %w", err)
	}
	return devices, nil
}

// ListPages fetches open pages/tabs from a device-specific port.
// The port comes from DeviceEntry.URL (e.g., "localhost:9222").
func ListPages(devicePort int) ([]webkit.PageEntry, error) {
	u := fmt.Sprintf("http://localhost:%d/json", devicePort)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("connecting to device on port %d: %w\nCheck the device port from 'list_devices' output", devicePort, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device on port %d returned HTTP %d", devicePort, resp.StatusCode)
	}

	var pages []webkit.PageEntry
	if err := json.NewDecoder(resp.Body).Decode(&pages); err != nil {
		return nil, fmt.Errorf("decoding page list: %w", err)
	}
	return pages, nil
}

// ListAllPages fetches pages from all connected devices.
// It first lists devices on port 9221, then queries each device's port for pages.
// If some devices fail but others succeed, partial results are returned.
// If ALL devices fail, a combined error is returned.
func ListAllPages() ([]webkit.PageEntry, error) {
	devices, err := ListDevices()
	if err != nil {
		return nil, err
	}

	var allPages []webkit.PageEntry
	var errs []error
	for _, d := range devices {
		port, err := DevicePort(d)
		if err != nil {
			errs = append(errs, fmt.Errorf("device %q: %w", d.DeviceName, err))
			continue
		}
		pages, err := ListPages(port)
		if err != nil {
			errs = append(errs, fmt.Errorf("device %q (port %d): %w", d.DeviceName, port, err))
			continue
		}
		allPages = append(allPages, pages...)
	}

	// If all devices failed, return a combined error.
	if len(allPages) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all devices failed: %w", errors.Join(errs...))
	}
	return allPages, nil
}

// DevicePort extracts the port number from a DeviceEntry's URL field.
// The URL is typically "localhost:9222".
func DevicePort(d webkit.DeviceEntry) (int, error) {
	if d.URL == "" {
		return 0, fmt.Errorf("device %q has no URL", d.DeviceName)
	}
	// Normalize: always add http:// so net/url can parse host:port correctly
	raw := d.URL
	if !hasScheme(raw) {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return 0, fmt.Errorf("parsing device URL %q: %w", d.URL, err)
	}
	port := u.Port()
	if port == "" {
		return 0, fmt.Errorf("device URL %q has no explicit port", d.URL)
	}
	var p int
	if _, err := fmt.Sscanf(port, "%d", &p); err != nil {
		return 0, fmt.Errorf("parsing port from %q: %w", d.URL, err)
	}
	return p, nil
}

// hasScheme checks if a string starts with a URI scheme (e.g., "http://").
// A scheme must start with a letter and be followed by "://".
func hasScheme(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ':':
			// Scheme requires "://" after the colon
			return i > 0 && i+2 < len(s) && s[i+1] == '/' && s[i+2] == '/'
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
			continue
		case i > 0 && ((c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.'):
			continue
		default:
			return false
		}
	}
	return false
}
