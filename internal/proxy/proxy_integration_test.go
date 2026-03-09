//go:build integration

package proxy

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func requireIWDP(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ios_webkit_debug_proxy"); err != nil {
		t.Skip("ios_webkit_debug_proxy not installed, skipping integration test")
	}
}

// stopIWDP kills any running ios_webkit_debug_proxy processes.
func stopIWDP(t *testing.T) {
	t.Helper()
	_ = exec.Command("pkill", "-f", "ios_webkit_debug_proxy").Run()
	// Wait for process to exit and port to free up.
	for i := 0; i < 10; i++ {
		if !IsRunning() {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestStart_Integration(t *testing.T) {
	requireIWDP(t)
	stopIWDP(t)
	t.Cleanup(func() { stopIWDP(t) })

	if IsRunning() {
		t.Fatal("expected iwdp to not be running after stopIWDP")
	}

	ctx := context.Background()
	if err := Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if !IsRunning() {
		t.Fatal("expected iwdp to be running after Start")
	}
}

func TestEnsureRunning_WhenStopped_Integration(t *testing.T) {
	requireIWDP(t)
	stopIWDP(t)
	t.Cleanup(func() { stopIWDP(t) })

	ctx := context.Background()
	if err := EnsureRunning(ctx); err != nil {
		t.Fatalf("EnsureRunning (from stopped): %v", err)
	}

	if !IsRunning() {
		t.Fatal("expected iwdp to be running after EnsureRunning")
	}
}

func TestEnsureRunning_WhenAlreadyRunning_Integration(t *testing.T) {
	requireIWDP(t)
	stopIWDP(t)
	t.Cleanup(func() { stopIWDP(t) })

	ctx := context.Background()
	// Start it first.
	if err := Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// EnsureRunning should be a no-op.
	if err := EnsureRunning(ctx); err != nil {
		t.Fatalf("EnsureRunning (already running): %v", err)
	}

	if !IsRunning() {
		t.Fatal("expected iwdp to still be running")
	}
}

func TestIsRunning_Integration(t *testing.T) {
	requireIWDP(t)
	stopIWDP(t)
	t.Cleanup(func() { stopIWDP(t) })

	if IsRunning() {
		t.Fatal("expected IsRunning=false when iwdp is stopped")
	}

	ctx := context.Background()
	if err := Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if !IsRunning() {
		t.Fatal("expected IsRunning=true after Start")
	}
}

func TestListDevices_Integration(t *testing.T) {
	requireIWDP(t)
	stopIWDP(t)
	t.Cleanup(func() { stopIWDP(t) })

	ctx := context.Background()
	if err := Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	devices, err := ListDevices()
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}

	// With no device connected, we may get an empty list or a single entry.
	// Either way it should not error.
	t.Logf("found %d device(s)", len(devices))
	for _, d := range devices {
		t.Logf("  %s (%s) at %s", d.DeviceName, d.DeviceID, d.URL)

		// Verify DevicePort works on real entries.
		port, err := DevicePort(d)
		if err != nil {
			t.Errorf("DevicePort(%q): %v", d.URL, err)
		} else {
			t.Logf("    port: %d", port)
		}
	}
}

func TestListAllPages_Integration(t *testing.T) {
	requireIWDP(t)
	stopIWDP(t)
	t.Cleanup(func() { stopIWDP(t) })

	ctx := context.Background()
	if err := Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	pages, err := ListAllPages()
	if err != nil {
		t.Fatalf("ListAllPages: %v", err)
	}

	t.Logf("found %d page(s) across all devices", len(pages))
	for _, p := range pages {
		t.Logf("  [%d] %s — %s", p.PageID, p.Title, p.URL)
		if p.WebSocketDebuggerURL != "" {
			t.Logf("       ws: %s", p.WebSocketDebuggerURL)
		}
	}
}

func TestListPages_WithRunningIWDP_Integration(t *testing.T) {
	requireIWDP(t)
	stopIWDP(t)
	t.Cleanup(func() { stopIWDP(t) })

	ctx := context.Background()
	if err := Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Query the first device port — even with no device connected,
	// iwdp may still respond (or return an error, which is fine).
	pages, err := ListPages(DefaultFirstDevicePort)
	if err != nil {
		// This is expected if no device is connected — not a test failure.
		t.Logf("ListPages on port %d returned error (expected without device): %v", DefaultFirstDevicePort, err)
		return
	}

	t.Logf("found %d page(s) on port %d", len(pages), DefaultFirstDevicePort)
	for _, p := range pages {
		t.Logf("  [%d] %s — %s", p.PageID, p.Title, p.URL)
	}
}

func TestStartStop_Lifecycle_Integration(t *testing.T) {
	requireIWDP(t)
	stopIWDP(t)
	t.Cleanup(func() { stopIWDP(t) })

	ctx := context.Background()

	// Start
	if err := Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !IsRunning() {
		t.Fatal("expected running after Start")
	}

	// List devices (should work)
	devices, err := ListDevices()
	if err != nil {
		t.Fatalf("ListDevices after Start: %v", err)
	}
	t.Logf("devices after start: %d", len(devices))

	// Stop
	stopIWDP(t)
	if IsRunning() {
		t.Fatal("expected not running after stop")
	}

	// ListDevices should fail now
	_, err = ListDevices()
	if err == nil {
		t.Fatal("expected error from ListDevices after stop")
	}

	// Restart via EnsureRunning
	if err := EnsureRunning(ctx); err != nil {
		t.Fatalf("EnsureRunning after stop: %v", err)
	}
	if !IsRunning() {
		t.Fatal("expected running after EnsureRunning")
	}
}
