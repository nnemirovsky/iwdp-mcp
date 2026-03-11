package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	cliBinaryOnce sync.Once
	cliBinaryPath string
	cliBinaryErr  error
)

func buildCLI(t *testing.T) string {
	t.Helper()
	cliBinaryOnce.Do(func() {
		dir, err := os.MkdirTemp("", "iwdp-e2e-cli-*")
		if err != nil {
			cliBinaryErr = err
			return
		}
		trackTempDir(dir)
		bin := filepath.Join(dir, "iwdp-cli")
		cmd := exec.Command("go", "build", "-o", bin, "../cmd/iwdp-cli")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			cliBinaryErr = err
			return
		}
		cliBinaryPath = bin
	})
	if cliBinaryErr != nil {
		t.Fatalf("building iwdp-cli: %v", cliBinaryErr)
	}
	return cliBinaryPath
}

func TestCLI_Help(t *testing.T) {
	bin := buildCLI(t)

	out, err := exec.Command(bin, "help").CombinedOutput()
	if err != nil {
		t.Fatalf("iwdp-cli help: %v\n%s", err, out)
	}

	output := string(out)
	expected := []string{
		"iwdp-cli",
		"Usage:",
		"devices",
		"pages",
		"eval",
		"navigate",
		"screenshot",
		"cookies",
		"dom",
		"console",
		"network",
	}
	for _, s := range expected {
		if !strings.Contains(output, s) {
			t.Errorf("help output missing %q", s)
		}
	}
}

func TestCLI_HelpFlag(t *testing.T) {
	bin := buildCLI(t)

	out, err := exec.Command(bin, "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("iwdp-cli --help: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Usage:") {
		t.Error("--help output missing Usage:")
	}
}

func TestCLI_HFlagShort(t *testing.T) {
	bin := buildCLI(t)

	out, err := exec.Command(bin, "-h").CombinedOutput()
	if err != nil {
		t.Fatalf("iwdp-cli -h: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Usage:") {
		t.Error("-h output missing Usage:")
	}
}

func TestCLI_NoArgs(t *testing.T) {
	bin := buildCLI(t)

	cmd := exec.Command(bin)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit code when called with no args")
	}
	if !strings.Contains(string(out), "Usage:") {
		t.Errorf("no-args output should show usage, got: %s", out)
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	bin := buildCLI(t)

	cmd := exec.Command(bin, "nonexistent")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit code for unknown command")
	}
	output := string(out)
	if !strings.Contains(output, "Unknown command") {
		t.Errorf("output should mention unknown command, got: %s", output)
	}
}

func TestCLI_EvalNoArgs(t *testing.T) {
	bin := buildCLI(t)

	cmd := exec.Command(bin, "eval")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit code for eval without expression")
	}
	if !strings.Contains(string(out), "Usage:") {
		t.Errorf("eval with no args should show usage, got: %s", out)
	}
}

func TestCLI_NavigateNoArgs(t *testing.T) {
	bin := buildCLI(t)

	cmd := exec.Command(bin, "navigate")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit code for navigate without URL")
	}
	if !strings.Contains(string(out), "Usage:") {
		t.Errorf("navigate with no args should show usage, got: %s", out)
	}
}

func TestCLI_DevicesWithoutIWDP(t *testing.T) {
	bin := buildCLI(t)

	// Devices command without iwdp running should give a helpful error.
	cmd := exec.Command(bin, "devices")
	out, err := cmd.CombinedOutput()
	if err == nil {
		// If iwdp happens to be running, that's fine too.
		t.Log("devices succeeded (iwdp may be running)")
		return
	}
	output := string(out)
	if !strings.Contains(output, "ios_webkit_debug_proxy") {
		t.Errorf("devices error should mention ios_webkit_debug_proxy, got: %s", output)
	}
}

func TestCLI_HelpCategories(t *testing.T) {
	bin := buildCLI(t)

	out, err := exec.Command(bin, "help").CombinedOutput()
	if err != nil {
		t.Fatalf("iwdp-cli help: %v\n%s", err, out)
	}
	output := string(out)

	categories := []string{
		"Core:", "Runtime:", "DOM:", "CSS:", "Interaction:",
		"Storage:", "Network:", "Console:", "Debugger:",
		"DOMDebugger:", "Performance:", "Animation:",
		"Canvas:", "LayerTree:", "Workers:", "Audit & Security:",
		"Browser:",
	}
	for _, cat := range categories {
		if !strings.Contains(output, cat) {
			t.Errorf("help output missing category %q", cat)
		}
	}
}

func TestCLI_HelpNewCommands(t *testing.T) {
	bin := buildCLI(t)

	out, err := exec.Command(bin, "help").CombinedOutput()
	if err != nil {
		t.Fatalf("iwdp-cli help: %v\n%s", err, out)
	}
	output := string(out)

	cmds := []string{
		"status", "restart-iwdp", "reload", "snapshot-node",
		"call-function", "get-properties",
		"query-selector", "query-selector-all", "get-outer-html",
		"get-attributes", "highlight-node",
		"get-matched-styles", "get-computed-style", "force-pseudo-state",
		"click", "fill", "type-text",
		"set-cookie", "delete-cookie", "get-local-storage", "get-session-storage",
		"get-response-body", "set-extra-headers", "disable-cache",
		"console-messages", "clear-console", "set-log-level",
		"debugger-enable", "set-breakpoint", "remove-breakpoint",
		"pause", "resume", "step-over", "step-into", "step-out",
		"get-script-source", "eval-on-frame",
		"set-dom-breakpoint", "set-event-breakpoint", "set-url-breakpoint",
		"timeline-record", "memory-track", "heap-snapshot", "heap-gc",
		"cpu-profile", "script-profile",
		"animation-enable", "animation-track", "get-animation-effect",
		"canvas-enable", "get-canvas-content", "get-shader-source",
		"get-layer-tree", "get-compositing-reasons",
		"worker-enable", "send-to-worker",
		"run-audit", "get-certificate-info",
		"browser-extensions-enable", "browser-extensions-disable",
	}
	for _, c := range cmds {
		if !strings.Contains(output, c) {
			t.Errorf("help output missing command %q", c)
		}
	}
}

func TestCLI_CommandsRequiringArgs(t *testing.T) {
	bin := buildCLI(t)

	// Commands that require arguments should exit non-zero and show Usage when called without them.
	cmds := []struct {
		name string
		args []string
	}{
		{"snapshot-node", nil},
		{"query-selector", nil},
		{"get-outer-html", nil},
		{"call-function", nil},
		{"get-properties", nil},
		{"query-selector-all", nil},
		{"get-attributes", nil},
		{"get-event-listeners", nil},
		{"highlight-node", nil},
		{"get-matched-styles", nil},
		{"get-computed-style", nil},
		{"get-inline-styles", nil},
		{"set-style-text", nil},
		{"get-stylesheet-text", nil},
		{"force-pseudo-state", nil},
		{"click", nil},
		{"fill", nil},
		{"type-text", nil},
		{"set-cookie", nil},
		{"delete-cookie", nil},
		{"get-local-storage", nil},
		{"set-local-storage-item", nil},
		{"remove-local-storage-item", nil},
		{"clear-local-storage", nil},
		{"get-session-storage", nil},
		{"set-session-storage-item", nil},
		{"remove-session-storage-item", nil},
		{"clear-session-storage", nil},
		{"list-indexed-databases", nil},
		{"get-indexed-db-data", nil},
		{"clear-indexed-db-store", nil},
		{"get-response-body", nil},
		{"set-extra-headers", nil},
		{"set-request-interception", nil},
		{"intercept-continue", nil},
		{"intercept-respond", nil},
		{"set-network-conditions", nil},
		{"disable-cache", nil},
		{"set-log-level", nil},
		{"set-breakpoint", nil},
		{"remove-breakpoint", nil},
		{"get-script-source", nil},
		{"search-in-content", nil},
		{"eval-on-frame", nil},
		{"set-pause-on-exceptions", nil},
		{"set-dom-breakpoint", nil},
		{"remove-dom-breakpoint", nil},
		{"set-event-breakpoint", nil},
		{"remove-event-breakpoint", nil},
		{"set-url-breakpoint", nil},
		{"remove-url-breakpoint", nil},
		{"get-animation-effect", nil},
		{"resolve-animation", nil},
		{"get-canvas-content", nil},
		{"start-canvas-recording", nil},
		{"stop-canvas-recording", nil},
		{"get-shader-source", nil},
		{"get-layer-tree", nil},
		{"get-compositing-reasons", nil},
		{"send-to-worker", nil},
		{"run-audit", nil},
		{"get-certificate-info", nil},
	}

	for _, tc := range cmds {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{tc.name}, tc.args...)
			cmd := exec.Command(bin, args...)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected non-zero exit for %s with no args", tc.name)
			}
			output := string(out)
			if !strings.Contains(output, "Usage:") {
				t.Errorf("%s with no args should show Usage, got: %s", tc.name, output)
			}
		})
	}
}
