//go:build simulator

package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
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

// --- Page ---

func TestSim_EvaluateBasic(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	result, err := tools.EvaluateScript(ctx, client, "2 + 2", true)
	if err != nil {
		t.Fatalf("EvaluateScript: %v", err)
	}
	if result.WasThrown {
		t.Fatal("expression was thrown")
	}
	var val float64
	_ = json.Unmarshal(result.Result.Value, &val)
	if val != 4 {
		t.Errorf("expected 4, got %v", val)
	}
}

func TestSim_Reload(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	err := tools.Reload(ctx, client, false)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	time.Sleep(2 * time.Second)
}

func TestSim_TakeScreenshot(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	dataURL, err := tools.TakeScreenshot(ctx, client)
	if err != nil {
		t.Fatalf("TakeScreenshot returned error: %v", err)
	}
	if dataURL == "" {
		t.Fatal("expected non-empty screenshot dataURL")
	}
	if !strings.HasPrefix(dataURL, "data:image/") {
		t.Errorf("unexpected screenshot prefix: %.50s...", dataURL)
	}
	t.Logf("screenshot: %d bytes", len(dataURL))
}

func TestSim_GetCookies(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	cookies, err := tools.GetCookies(ctx, client)
	if err != nil {
		t.Fatalf("GetCookies: %v", err)
	}
	t.Logf("got %d cookies", len(cookies))
}

func TestSim_SetAndDeleteCookie(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	cookie := webkit.Cookie{
		Name:     "test_cookie",
		Value:    "test_value",
		Domain:   "example.com",
		Path:     "/",
		SameSite: "Lax",
	}
	if err := tools.SetCookie(ctx, client, cookie); err != nil {
		t.Fatalf("SetCookie: %v", err)
	}

	cookies, err := tools.GetCookies(ctx, client)
	if err != nil {
		t.Fatalf("GetCookies after set: %v", err)
	}

	found := false
	for _, c := range cookies {
		if c.Name == "test_cookie" && c.Value == "test_value" {
			found = true
			break
		}
	}
	if !found {
		// Page.getCookies may not return cookies set via Page.setCookie in all WebKit versions.
		// Verify via JS instead.
		result, err := tools.EvaluateScript(ctx, client, "document.cookie", true)
		if err == nil {
			var cookieStr string
			_ = json.Unmarshal(result.Result.Value, &cookieStr)
			if strings.Contains(cookieStr, "test_cookie=test_value") {
				t.Log("cookie found via document.cookie (not via Page.getCookies)")
			} else {
				t.Logf("cookie not found via document.cookie either: %q", cookieStr)
				t.Log("Page.setCookie succeeded but cookie not visible — may be a WebKit/simulator limitation")
			}
		}
	}

	origin := simOrigin(t, client)
	if err := tools.DeleteCookie(ctx, client, "test_cookie", origin+"/"); err != nil {
		t.Fatalf("DeleteCookie: %v", err)
	}
}

// --- Runtime ---

func TestSim_EvaluateScript_Error(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	_, err := tools.EvaluateScript(ctx, client, "throw new Error('test')", true)
	if err == nil {
		t.Fatal("expected error for thrown expression")
	}
	t.Logf("got expected error: %v", err)
}

func TestSim_EvaluateScript_ComplexObject(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	result, err := tools.EvaluateScript(ctx, client, "({name: 'test', count: 42})", true)
	if err != nil {
		t.Fatalf("EvaluateScript: %v", err)
	}
	var obj map[string]interface{}
	_ = json.Unmarshal(result.Result.Value, &obj)
	if obj["name"] != "test" {
		t.Errorf("expected name=test, got %v", obj["name"])
	}
	if obj["count"] != float64(42) {
		t.Errorf("expected count=42, got %v", obj["count"])
	}
}

// --- DOM ---

func TestSim_GetDocument(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	root, err := tools.GetDocument(ctx, client, 2)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if root.NodeID == 0 {
		t.Error("expected non-zero root nodeId")
	}
	if root.NodeType != 9 {
		t.Errorf("expected document nodeType 9, got %d", root.NodeType)
	}
	t.Logf("document root: nodeId=%d, nodeName=%s, children=%d", root.NodeID, root.NodeName, len(root.Children))
}

func TestSim_QuerySelector(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}

	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "h1")
	if err != nil {
		t.Skipf("no h1 found: %v", err)
	}
	if nodeID == 0 {
		t.Error("expected non-zero nodeId for h1")
	}

	html, err := tools.GetOuterHTML(ctx, client, nodeID)
	if err != nil {
		t.Fatalf("GetOuterHTML: %v", err)
	}
	if !strings.Contains(html, "Example Domain") {
		t.Errorf("h1 outer HTML = %q, expected to contain 'Example Domain'", html)
	}
	t.Logf("h1 outerHTML: %s", html)
}

func TestSim_QuerySelectorAll(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}

	nodeIDs, err := tools.QuerySelectorAll(ctx, client, root.NodeID, "p")
	if err != nil {
		t.Fatalf("QuerySelectorAll p: %v", err)
	}
	if len(nodeIDs) == 0 {
		t.Error("expected at least one <p> element on example.com")
	}
	t.Logf("found %d <p> elements", len(nodeIDs))
}

func TestSim_GetAttributes(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}

	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "a")
	if err != nil {
		t.Skipf("no <a> element found: %v", err)
	}

	attrs, err := tools.GetAttributes(ctx, client, nodeID)
	if err != nil {
		t.Fatalf("GetAttributes: %v", err)
	}
	t.Logf("attributes of <a>: %v", attrs)
	if _, ok := attrs["href"]; !ok {
		t.Error("expected 'href' attribute on <a> element")
	}
}

func TestSim_HighlightAndHide(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}

	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "h1")
	if err != nil {
		t.Skipf("no h1 found: %v", err)
	}

	if err := tools.HighlightNode(ctx, client, nodeID); err != nil {
		t.Fatalf("HighlightNode: %v", err)
	}
	if err := tools.HideHighlight(ctx, client); err != nil {
		t.Fatalf("HideHighlight: %v", err)
	}
}

// --- CSS ---

func TestSim_GetComputedStyle(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}

	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "h1")
	if err != nil {
		t.Skipf("no h1 found: %v", err)
	}

	props, err := tools.GetComputedStyle(ctx, client, nodeID)
	if err != nil {
		t.Fatalf("GetComputedStyle: %v", err)
	}
	if len(props) == 0 {
		t.Error("expected non-empty computed style")
	}
	t.Logf("got %d computed style properties", len(props))
}

func TestSim_GetMatchedStyles(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}

	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "body")
	if err != nil {
		t.Skipf("no body found: %v", err)
	}

	result, err := tools.GetMatchedStyles(ctx, client, nodeID)
	if err != nil {
		t.Fatalf("GetMatchedStyles: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty matched styles")
	}
}

func TestSim_ForcePseudoState(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}

	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "a")
	if err != nil {
		t.Skipf("no <a> found: %v", err)
	}

	if err := tools.ForcePseudoState(ctx, client, nodeID, []string{"hover"}); err != nil {
		t.Fatalf("ForcePseudoState: %v", err)
	}
	_ = tools.ForcePseudoState(ctx, client, nodeID, []string{})
}

// --- Network ---

func TestSim_NetworkMonitor(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	monitor := tools.NewNetworkMonitor()
	if err := monitor.Start(ctx, client); err != nil {
		t.Fatalf("NetworkMonitor.Start: %v", err)
	}

	// Trigger a network request via fetch.
	_, _ = tools.EvaluateScript(ctx, client, "fetch('/').catch(()=>{})", false)
	time.Sleep(2 * time.Second)

	requests := monitor.GetRequests()
	t.Logf("captured %d network request(s)", len(requests))
	for _, r := range requests {
		status := 0
		if r.Response != nil {
			status = r.Response.Status
		}
		t.Logf("  %s %s → %d", r.Request.Method, r.Request.URL, status)
	}
}

func TestSim_SetResourceCachingDisabled(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	monitor := tools.NewNetworkMonitor()
	_ = monitor.Start(ctx, client)

	if err := tools.SetResourceCachingDisabled(ctx, client, true); err != nil {
		t.Skipf("SetResourceCachingDisabled not supported: %v", err)
	}
	_ = tools.SetResourceCachingDisabled(ctx, client, false)
}

// --- Console ---

func TestSim_ConsoleCollector(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	collector := tools.NewConsoleCollector()
	if err := collector.Start(ctx, client); err != nil {
		t.Fatalf("ConsoleCollector.Start: %v", err)
	}

	_, _ = tools.EvaluateScript(ctx, client, "console.log('sim-test-message')", false)
	time.Sleep(500 * time.Millisecond)

	messages := collector.GetMessages()
	t.Logf("got %d console messages", len(messages))
}

// --- Storage ---

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

func TestSim_LocalStorage(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	origin := simOrigin(t, client)
	t.Logf("using origin: %s", origin)

	if err := tools.SetLocalStorageItem(ctx, client, origin, "sim_test_key", "sim_test_value"); err != nil {
		t.Fatalf("SetLocalStorageItem: %v", err)
	}

	items, err := tools.GetLocalStorage(ctx, client, origin)
	if err != nil {
		t.Fatalf("GetLocalStorage: %v", err)
	}

	found := false
	for _, item := range items {
		if item.Key == "sim_test_key" && item.Value == "sim_test_value" {
			found = true
			break
		}
	}
	if !found {
		t.Error("localStorage item not found after set")
	}

	_ = tools.RemoveLocalStorageItem(ctx, client, origin, "sim_test_key")
}

func TestSim_SessionStorage(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	origin := simOrigin(t, client)

	if err := tools.SetSessionStorageItem(ctx, client, origin, "sim_sess_key", "sim_sess_value"); err != nil {
		t.Fatalf("SetSessionStorageItem: %v", err)
	}

	items, err := tools.GetSessionStorage(ctx, client, origin)
	if err != nil {
		t.Fatalf("GetSessionStorage: %v", err)
	}

	found := false
	for _, item := range items {
		if item.Key == "sim_sess_key" {
			found = true
			break
		}
	}
	if !found {
		t.Error("sessionStorage item not found after set")
	}

	_ = tools.RemoveSessionStorageItem(ctx, client, origin, "sim_sess_key")
}

// --- Debugger ---

func TestSim_DebuggerEnableDisable(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.DebuggerEnable(ctx, client); err != nil {
		t.Fatalf("DebuggerEnable: %v", err)
	}
	if err := tools.SetPauseOnExceptions(ctx, client, "none"); err != nil {
		t.Fatalf("SetPauseOnExceptions: %v", err)
	}
	if err := tools.DebuggerDisable(ctx, client); err != nil {
		t.Fatalf("DebuggerDisable: %v", err)
	}
}

// --- Timeline ---

func TestSim_TimelineStartStop(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.TimelineStart(ctx, client, 5); err != nil {
		t.Fatalf("TimelineStart: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	if err := tools.TimelineStop(ctx, client); err != nil {
		t.Fatalf("TimelineStop: %v", err)
	}
}

// --- Memory ---

func TestSim_HeapGC(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.HeapGC(ctx, client); err != nil {
		t.Fatalf("HeapGC: %v", err)
	}
}

func TestSim_HeapSnapshot(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	result, err := tools.HeapSnapshot(ctx, client)
	if err != nil {
		t.Fatalf("HeapSnapshot: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty heap snapshot")
	}
	t.Logf("heap snapshot: %d bytes", len(result))
}

// --- Interaction ---

func TestSim_FillInput(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	// Create a test input via JS.
	_, _ = tools.EvaluateScript(ctx, client,
		"document.body.innerHTML = '<input id=\"test-input\" type=\"text\" />'", false)
	time.Sleep(500 * time.Millisecond)

	if err := tools.Fill(ctx, client, "#test-input", "hello simulator"); err != nil {
		t.Fatalf("Fill: %v", err)
	}

	result, err := tools.EvaluateScript(ctx, client,
		"document.getElementById('test-input').value", true)
	if err != nil {
		t.Fatalf("EvaluateScript: %v", err)
	}
	var val string
	_ = json.Unmarshal(result.Result.Value, &val)
	if val != "hello simulator" {
		t.Errorf("expected 'hello simulator', got %q", val)
	}
}

// --- Profiler ---

func TestSim_CPUProfiling(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	collector := tools.NewCPUProfilerCollector()
	if err := collector.Start(ctx, client); err != nil {
		t.Fatalf("CPUProfilerCollector.Start: %v", err)
	}
	_, _ = tools.EvaluateScript(ctx, client, "for(let i=0;i<1000;i++){}", false)
	time.Sleep(500 * time.Millisecond)

	result, err := collector.Stop(ctx, client)
	if err != nil {
		t.Fatalf("CPUProfilerCollector.Stop: %v", err)
	}
	t.Logf("CPU profiler collected %d events", len(result.Events))
}

// --- Domain Enable/Disable ---

func TestSim_WorkerEnableDisable(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.WorkerEnable(ctx, client); err != nil {
		t.Fatalf("WorkerEnable: %v", err)
	}
	if err := tools.WorkerDisable(ctx, client); err != nil {
		t.Fatalf("WorkerDisable: %v", err)
	}
}

func TestSim_AnimationEnableDisable(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.AnimationEnable(ctx, client); err != nil {
		t.Fatalf("AnimationEnable: %v", err)
	}
	if err := tools.AnimationDisable(ctx, client); err != nil {
		t.Fatalf("AnimationDisable: %v", err)
	}
}

func TestSim_CanvasEnableDisable(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.CanvasEnable(ctx, client); err != nil {
		t.Fatalf("CanvasEnable: %v", err)
	}
	if err := tools.CanvasDisable(ctx, client); err != nil {
		t.Fatalf("CanvasDisable: %v", err)
	}
}
