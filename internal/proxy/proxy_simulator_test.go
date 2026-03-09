//go:build simulator

package proxy

import (
	"os"
	"testing"
)

// Simulator tests expect ios_webkit_debug_proxy to already be running
// (started by scripts/sim-setup.sh). They verify the proxy package works
// against a real simulator.

func requireSimEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("IWDP_SIM_WS_URL") == "" {
		t.Skip("IWDP_SIM_WS_URL not set — run scripts/sim-setup.sh first")
	}
}

func TestSim_IsRunning(t *testing.T) {
	requireSimEnv(t)

	if !IsRunning() {
		t.Fatal("expected iwdp to be running (started by sim-setup.sh)")
	}
}

func TestSim_ListDevices(t *testing.T) {
	requireSimEnv(t)

	devices, err := ListDevices()
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) == 0 {
		t.Fatal("expected at least one device (the simulator)")
	}

	for _, d := range devices {
		t.Logf("device: %s (%s) at %s", d.DeviceName, d.DeviceID, d.URL)
	}

	// Simulator typically shows as "SIMULATOR".
	found := false
	for _, d := range devices {
		if d.DeviceName == "SIMULATOR" || d.DeviceID == "SIMULATOR" {
			found = true
			break
		}
	}
	if !found {
		t.Log("warning: no device named SIMULATOR — may be a physical device")
	}
}

func TestSim_ListPages(t *testing.T) {
	requireSimEnv(t)

	pages, err := ListPages(DefaultFirstDevicePort)
	if err != nil {
		t.Fatalf("ListPages: %v", err)
	}
	if len(pages) == 0 {
		t.Fatal("expected at least one page from simulator Safari")
	}

	for _, p := range pages {
		t.Logf("page: [%d] %s — %s (ws: %s)", p.PageID, p.Title, p.URL, p.WebSocketDebuggerURL)
	}
}

func TestSim_ListAllPages(t *testing.T) {
	requireSimEnv(t)

	pages, err := ListAllPages()
	if err != nil {
		t.Fatalf("ListAllPages: %v", err)
	}
	if len(pages) == 0 {
		t.Fatal("expected at least one page")
	}
	t.Logf("found %d page(s) across all devices", len(pages))
}

func TestSim_DevicePort(t *testing.T) {
	requireSimEnv(t)

	devices, err := ListDevices()
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) == 0 {
		t.Skip("no devices")
	}

	for _, d := range devices {
		port, err := DevicePort(d)
		if err != nil {
			t.Errorf("DevicePort(%q): %v", d.URL, err)
			continue
		}
		if port < 9222 {
			t.Errorf("unexpected port %d for device %s", port, d.DeviceName)
		}
		t.Logf("device %s → port %d", d.DeviceName, port)
	}
}
