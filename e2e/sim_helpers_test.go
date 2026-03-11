//go:build simulator

package e2e_test

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// shared simulator connection — WebKit only allows one debugger connection per page.
var (
	simOnce   sync.Once
	simClient *webkit.Client
	simErr    error
)

func getSimClient(t *testing.T) *webkit.Client {
	t.Helper()
	wsURL := os.Getenv("IWDP_SIM_WS_URL")
	if wsURL == "" {
		t.Skip("IWDP_SIM_WS_URL not set — run scripts/sim-setup.sh first")
	}

	simOnce.Do(func() {
		// CI runners are slower — give iwdp more time to send Target.targetCreated.
		webkit.TargetWaitTimeout = 2 * time.Second
		simClient, simErr = webkit.NewClient(context.Background(), wsURL)
	})
	if simErr != nil {
		t.Fatalf("connecting to simulator: %v", simErr)
	}
	return simClient
}

func simCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

func simOrigin(t *testing.T, client *webkit.Client) string {
	t.Helper()
	ctx, cancel := simCtx()
	defer cancel()
	result, err := tools.EvaluateScript(ctx, client, "window.location.origin", true)
	if err != nil {
		t.Fatalf("getting origin: %v", err)
	}
	var origin string
	_ = json.Unmarshal(result.Result.Value, &origin)
	return origin
}
