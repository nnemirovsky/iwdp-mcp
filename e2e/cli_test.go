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
