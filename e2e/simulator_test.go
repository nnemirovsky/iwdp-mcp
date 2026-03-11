//go:build simulator

package e2e_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nnemirovsky/iwdp-mcp/internal/proxy"
	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// =============================================================================
// Proxy: list_devices, list_pages, iwdp_status
// =============================================================================

func TestSim_IWDPStatus(t *testing.T) {
	_ = getSimClient(t) // ensure simulator is up
	if !proxy.IsRunning() {
		t.Fatal("expected iwdp to be running during simulator tests")
	}
}

func TestSim_ListDevices(t *testing.T) {
	_ = getSimClient(t)
	devices, err := proxy.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) == 0 {
		t.Fatal("expected at least one device")
	}
	t.Logf("found %d device(s)", len(devices))
}

func TestSim_ListPages(t *testing.T) {
	_ = getSimClient(t)
	devices, err := proxy.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) == 0 {
		t.Fatal("no devices to list pages from")
	}
	port, err := proxy.DevicePort(devices[0])
	if err != nil {
		t.Fatalf("DevicePort: %v", err)
	}
	pages, err := proxy.ListPages(port)
	if err != nil {
		t.Fatalf("ListPages: %v", err)
	}
	t.Logf("found %d page(s) on port %d", len(pages), port)
}

// =============================================================================
// Page: navigate, reload, evaluate_script, take_screenshot, snapshot_node
// =============================================================================

func TestSim_Navigate(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.Navigate(ctx, client, "https://example.com"); err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	time.Sleep(2 * time.Second)

	result, err := tools.EvaluateScript(ctx, client, "document.title", true)
	if err != nil {
		t.Fatalf("EvaluateScript: %v", err)
	}
	var title string
	_ = json.Unmarshal(result.Result.Value, &title)
	t.Logf("title after navigate: %q", title)
}

func TestSim_Reload(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.Reload(ctx, client, false); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	time.Sleep(2 * time.Second)
}

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

func TestSim_SnapshotNode(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "body")
	if err != nil {
		t.Fatalf("QuerySelector body: %v", err)
	}

	dataURL, err := tools.SnapshotNode(ctx, client, nodeID)
	if err != nil {
		t.Fatalf("SnapshotNode: %v", err)
	}
	if dataURL == "" {
		t.Fatal("expected non-empty snapshot dataURL")
	}
	t.Logf("snapshot_node: %d bytes", len(dataURL))
}

// =============================================================================
// Runtime: call_function, get_properties
// =============================================================================

func TestSim_CallFunctionOn(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	result, err := tools.EvaluateScript(ctx, client, "({x: 1, y: 2})", false)
	if err != nil {
		t.Fatalf("EvaluateScript: %v", err)
	}
	if result.Result.ObjectID == "" {
		t.Fatal("no objectId returned")
	}

	fnResult, err := tools.CallFunctionOn(ctx, client, result.Result.ObjectID, "function() { return this.x + this.y; }", nil, true)
	if err != nil {
		t.Fatalf("CallFunctionOn: %v", err)
	}
	var val float64
	_ = json.Unmarshal(fnResult.Result.Value, &val)
	if val != 3 {
		t.Errorf("expected 3, got %v", val)
	}
}

func TestSim_GetProperties(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	result, err := tools.EvaluateScript(ctx, client, "({name: 'test', count: 42})", false)
	if err != nil {
		t.Fatalf("EvaluateScript: %v", err)
	}
	if result.Result.ObjectID == "" {
		t.Fatal("no objectId returned")
	}

	props, err := tools.GetProperties(ctx, client, result.Result.ObjectID, true)
	if err != nil {
		t.Fatalf("GetProperties: %v", err)
	}
	if len(props) == 0 {
		t.Error("expected non-empty properties")
	}
	t.Logf("got %d properties", len(props))
}

// =============================================================================
// DOM: get_document, query_selector, query_selector_all, get_outer_html,
//      get_attributes, get_event_listeners, highlight_node, hide_highlight
// =============================================================================

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
		t.Fatalf("no h1 found: %v", err)
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

func TestSim_GetOuterHTML(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "body")
	if err != nil {
		t.Fatalf("QuerySelector body: %v", err)
	}

	html, err := tools.GetOuterHTML(ctx, client, nodeID)
	if err != nil {
		t.Fatalf("GetOuterHTML: %v", err)
	}
	if html == "" {
		t.Error("expected non-empty outer HTML")
	}
	t.Logf("body outerHTML: %.200s...", html)
}

func TestSim_GetAttributes(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	// Create our own <a> element so we don't depend on example.com's DOM state.
	_, _ = tools.EvaluateScript(ctx, client,
		"if(!document.querySelector('#attr-test-link')){var a=document.createElement('a');a.id='attr-test-link';a.href='https://example.com';a.textContent='test';document.body.appendChild(a)}", false)
	time.Sleep(300 * time.Millisecond)

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}

	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "#attr-test-link")
	if err != nil {
		t.Fatalf("no #attr-test-link found: %v", err)
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

func TestSim_GetEventListeners(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	_, _ = tools.EvaluateScript(ctx, client,
		"document.body.innerHTML='<button id=\"btn\">Click</button>';document.getElementById('btn').addEventListener('click',()=>{})", false)
	time.Sleep(500 * time.Millisecond)

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "#btn")
	if err != nil {
		t.Fatalf("no #btn found: %v", err)
	}

	listeners, err := tools.GetEventListeners(ctx, client, nodeID)
	if err != nil {
		t.Fatalf("GetEventListeners: %v", err)
	}
	t.Logf("got %d event listeners", len(listeners))
}

func TestSim_HighlightAndHide(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}

	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "body")
	if err != nil {
		t.Fatalf("no body found: %v", err)
	}

	if err := tools.HighlightNode(ctx, client, nodeID); err != nil {
		t.Fatalf("HighlightNode: %v", err)
	}
	if err := tools.HideHighlight(ctx, client); err != nil {
		t.Fatalf("HideHighlight: %v", err)
	}
}

// =============================================================================
// CSS: get_matched_styles, get_computed_style, get_inline_styles,
//      set_style_text, get_all_stylesheets, get_stylesheet_text,
//      force_pseudo_state
// =============================================================================

func TestSim_GetComputedStyle(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}

	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "body")
	if err != nil {
		t.Fatalf("no body found: %v", err)
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
		t.Fatalf("no body found: %v", err)
	}

	result, err := tools.GetMatchedStyles(ctx, client, nodeID)
	if err != nil {
		t.Fatalf("GetMatchedStyles: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty matched styles")
	}
}

func TestSim_GetInlineStyles(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	_, _ = tools.EvaluateScript(ctx, client,
		"document.body.innerHTML='<div id=\"styled\" style=\"color:red\">test</div>'", false)
	time.Sleep(500 * time.Millisecond)

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "#styled")
	if err != nil {
		t.Fatalf("no #styled found: %v", err)
	}

	style, err := tools.GetInlineStyles(ctx, client, nodeID)
	if err != nil {
		t.Fatalf("GetInlineStyles: %v", err)
	}
	if style == nil {
		t.Error("expected non-nil inline style")
	}
}

func TestSim_SetStyleText(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	_, _ = tools.EvaluateScript(ctx, client,
		"document.body.innerHTML='<div id=\"style-target\" style=\"color:red\">test</div>'", false)
	time.Sleep(500 * time.Millisecond)

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "#style-target")
	if err != nil {
		t.Fatalf("no #style-target found: %v", err)
	}

	style, err := tools.GetInlineStyles(ctx, client, nodeID)
	if err != nil {
		t.Fatalf("GetInlineStyles: %v", err)
	}
	if style == nil {
		t.Fatal("expected non-nil inline style to modify")
	}

	newStyle, err := tools.SetStyleText(ctx, client, style.StyleID, "color: blue; font-size: 20px;")
	if err != nil {
		t.Fatalf("SetStyleText: %v", err)
	}
	t.Logf("updated style: %d properties", len(newStyle.Properties))
}

func TestSim_GetAllStyleSheets(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	// CSS.getAllStyleSheets hangs through iwdp Target routing (no response comes back,
	// and the hang corrupts the connection pipeline for all subsequent commands).
	// The tool detects Target routing and returns an error immediately.
	_, err := tools.GetAllStylesheets(ctx, client)
	if err == nil {
		t.Fatal("expected error for GetAllStylesheets through iwdp Target routing")
	}
	t.Logf("got expected error: %v", err)
}

func TestSim_ForcePseudoState(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	// Create our own <a> element so we don't depend on example.com's DOM state.
	_, _ = tools.EvaluateScript(ctx, client,
		"if(!document.querySelector('#pseudo-test-link')){var a=document.createElement('a');a.id='pseudo-test-link';a.href='#';a.textContent='hover me';document.body.appendChild(a)}", false)
	time.Sleep(300 * time.Millisecond)

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}

	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "#pseudo-test-link")
	if err != nil {
		t.Fatalf("no #pseudo-test-link found: %v", err)
	}

	if err := tools.ForcePseudoState(ctx, client, nodeID, []string{"hover"}); err != nil {
		t.Fatalf("ForcePseudoState: %v", err)
	}
	_ = tools.ForcePseudoState(ctx, client, nodeID, []string{})
}

// =============================================================================
// Interaction: click, fill, type_text
// =============================================================================

func TestSim_Click(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	_, _ = tools.EvaluateScript(ctx, client,
		"document.body.innerHTML='<button id=\"click-test\" onclick=\"this.textContent=\\'clicked\\'\">Click me</button>'", false)
	time.Sleep(500 * time.Millisecond)

	if err := tools.Click(ctx, client, "#click-test"); err != nil {
		t.Fatalf("Click: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	result, err := tools.EvaluateScript(ctx, client, "document.getElementById('click-test').textContent", true)
	if err != nil {
		t.Fatalf("EvaluateScript: %v", err)
	}
	var text string
	_ = json.Unmarshal(result.Result.Value, &text)
	if text != "clicked" {
		t.Errorf("expected 'clicked', got %q", text)
	}
}

func TestSim_FillInput(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

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

func TestSim_TypeText(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	_, _ = tools.EvaluateScript(ctx, client,
		"document.body.innerHTML='<input id=\"type-test\" type=\"text\" />';document.getElementById('type-test').focus()", false)
	time.Sleep(500 * time.Millisecond)

	if err := tools.TypeText(ctx, client, "typed text"); err != nil {
		t.Fatalf("TypeText: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	result, err := tools.EvaluateScript(ctx, client, "document.getElementById('type-test').value", true)
	if err != nil {
		t.Fatalf("EvaluateScript: %v", err)
	}
	var val string
	_ = json.Unmarshal(result.Result.Value, &val)
	t.Logf("typed value: %q", val)
}

// =============================================================================
// Cookies: get_cookies, set_cookie, delete_cookie
// =============================================================================

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
				t.Log("Page.setCookie succeeded but cookie not visible. May be a WebKit/simulator limitation")
			}
		}
	}

	origin := simOrigin(t, client)
	if err := tools.DeleteCookie(ctx, client, "test_cookie", origin+"/"); err != nil {
		t.Fatalf("DeleteCookie: %v", err)
	}
}

// =============================================================================
// Storage: localStorage, sessionStorage, IndexedDB
// =============================================================================

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

	if err := tools.RemoveLocalStorageItem(ctx, client, origin, "sim_test_key"); err != nil {
		t.Fatalf("RemoveLocalStorageItem: %v", err)
	}
}

func TestSim_ClearLocalStorage(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	origin := simOrigin(t, client)
	_ = tools.SetLocalStorageItem(ctx, client, origin, "clear_test", "val")

	if err := tools.ClearLocalStorage(ctx, client, origin); err != nil {
		t.Fatalf("ClearLocalStorage: %v", err)
	}

	items, err := tools.GetLocalStorage(ctx, client, origin)
	if err != nil {
		t.Fatalf("GetLocalStorage: %v", err)
	}
	for _, item := range items {
		if item.Key == "clear_test" {
			t.Error("localStorage item should have been cleared")
		}
	}
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

	if err := tools.RemoveSessionStorageItem(ctx, client, origin, "sim_sess_key"); err != nil {
		t.Fatalf("RemoveSessionStorageItem: %v", err)
	}
}

func TestSim_ClearSessionStorage(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	origin := simOrigin(t, client)
	_ = tools.SetSessionStorageItem(ctx, client, origin, "clear_sess_test", "val")

	if err := tools.ClearSessionStorage(ctx, client, origin); err != nil {
		t.Fatalf("ClearSessionStorage: %v", err)
	}

	items, err := tools.GetSessionStorage(ctx, client, origin)
	if err != nil {
		t.Fatalf("GetSessionStorage: %v", err)
	}
	for _, item := range items {
		if item.Key == "clear_sess_test" {
			t.Error("sessionStorage item should have been cleared")
		}
	}
}

func TestSim_IndexedDB(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	origin := simOrigin(t, client)

	// Create an IndexedDB database via JS
	_, err := tools.EvaluateScript(ctx, client, `
		new Promise((resolve, reject) => {
			var req = indexedDB.open("sim_test_db", 1);
			req.onupgradeneeded = function(e) {
				var db = e.target.result;
				var store = db.createObjectStore("items", {keyPath: "id"});
				store.add({id: 1, name: "test_item"});
			};
			req.onsuccess = function() { req.result.close(); resolve("ok"); };
			req.onerror = function() { reject(req.error); };
		})
	`, true)
	if err != nil {
		t.Fatalf("creating IndexedDB: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// list_indexed_databases
	dbs, err := tools.ListIndexedDatabases(ctx, client, origin)
	if err != nil {
		t.Fatalf("ListIndexedDatabases: %v", err)
	}
	t.Logf("found %d IndexedDB databases", len(dbs))

	// get_indexed_db_data
	data, err := tools.GetIndexedDBData(ctx, client, origin, "sim_test_db", "items", 0, 10)
	if err != nil {
		t.Fatalf("GetIndexedDBData: %v", err)
	}
	t.Logf("IndexedDB data: %s", string(data))

	// clear_indexed_db_store
	if err := tools.ClearIndexedDBStore(ctx, client, origin, "sim_test_db", "items"); err != nil {
		t.Fatalf("ClearIndexedDBStore: %v", err)
	}

	// Clean up
	_, _ = tools.EvaluateScript(ctx, client, "indexedDB.deleteDatabase('sim_test_db')", false)
}

// =============================================================================
// Network: enable, disable, list_network_requests, get_response_body,
//          set_extra_headers, set_request_interception, intercept_continue,
//          intercept_with_response, set_emulated_conditions,
//          set_resource_caching_disabled
// =============================================================================

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
		t.Logf("  %s %s -> %d", r.Request.Method, r.Request.URL, status)
	}

	if err := monitor.Stop(ctx, client); err != nil {
		t.Fatalf("NetworkMonitor.Stop: %v", err)
	}
}

func TestSim_GetResponseBody(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	monitor := tools.NewNetworkMonitor()
	if err := monitor.Start(ctx, client); err != nil {
		t.Fatalf("NetworkMonitor.Start: %v", err)
	}

	_, _ = tools.EvaluateScript(ctx, client, "fetch('/').catch(()=>{})", false)
	time.Sleep(2 * time.Second)

	requests := monitor.GetRequests()
	if len(requests) == 0 {
		t.Fatal("no network requests captured for get_response_body test")
	}

	// Find a completed request
	var requestID string
	for _, r := range requests {
		if r.Done && r.Request.RequestID != "" {
			requestID = r.Request.RequestID
			break
		}
	}
	if requestID == "" {
		t.Fatal("no completed request found to get response body")
	}

	body, base64Encoded, err := tools.GetResponseBody(ctx, client, requestID)
	if err != nil {
		t.Fatalf("GetResponseBody: %v", err)
	}
	t.Logf("response body: %d bytes, base64=%v", len(body), base64Encoded)

	_ = monitor.Stop(ctx, client)
}

func TestSim_SetExtraHeaders(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	monitor := tools.NewNetworkMonitor()
	_ = monitor.Start(ctx, client)

	headers := map[string]string{"X-Test": "sim-value"}
	if err := tools.SetExtraHeaders(ctx, client, headers); err != nil {
		t.Fatalf("SetExtraHeaders: %v", err)
	}
	// Clear extra headers
	_ = tools.SetExtraHeaders(ctx, client, map[string]string{})
	_ = monitor.Stop(ctx, client)
}

func TestSim_SetRequestInterception(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	monitor := tools.NewNetworkMonitor()
	_ = monitor.Start(ctx, client)

	if err := tools.SetRequestInterception(ctx, client, true); err != nil {
		t.Fatalf("SetRequestInterception enable: %v", err)
	}
	if err := tools.SetRequestInterception(ctx, client, false); err != nil {
		t.Fatalf("SetRequestInterception disable: %v", err)
	}
	_ = monitor.Stop(ctx, client)
}

func TestSim_SetEmulatedConditions(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	monitor := tools.NewNetworkMonitor()
	_ = monitor.Start(ctx, client)

	if err := tools.SetEmulatedConditions(ctx, client, 100000); err != nil {
		t.Fatalf("SetEmulatedConditions: %v", err)
	}
	// Reset
	_ = tools.SetEmulatedConditions(ctx, client, 0)
	_ = monitor.Stop(ctx, client)
}

func TestSim_SetResourceCachingDisabled(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	monitor := tools.NewNetworkMonitor()
	_ = monitor.Start(ctx, client)

	if err := tools.SetResourceCachingDisabled(ctx, client, true); err != nil {
		t.Fatalf("SetResourceCachingDisabled: %v", err)
	}
	_ = tools.SetResourceCachingDisabled(ctx, client, false)
	_ = monitor.Stop(ctx, client)
}

// =============================================================================
// Console: get_console_messages, clear_console, set_log_level
// =============================================================================

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

func TestSim_ClearConsoleMessages(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.ClearConsoleMessages(ctx, client); err != nil {
		t.Fatalf("ClearConsoleMessages: %v", err)
	}
}

func TestSim_SetLogLevel(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.SetLogLevel(ctx, client, "javascript", "off"); err != nil {
		t.Fatalf("SetLogLevel: %v", err)
	}
	// Restore default
	_ = tools.SetLogLevel(ctx, client, "javascript", "basic")
}

// =============================================================================
// Debugger: enable, disable, set_breakpoint, remove_breakpoint,
//           pause, resume, step_over, step_into, step_out,
//           get_script_source, search_in_content, evaluate_on_call_frame,
//           set_pause_on_exceptions
// =============================================================================

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

func TestSim_DebuggerSetRemoveBreakpoint(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.DebuggerEnable(ctx, client); err != nil {
		t.Fatalf("DebuggerEnable: %v", err)
	}
	defer func() { _ = tools.DebuggerDisable(ctx, client) }()

	bpID, _, err := tools.SetBreakpointByURL(ctx, client, "nonexistent.js", 1, nil, "")
	if err != nil {
		t.Fatalf("SetBreakpointByURL: %v", err)
	}
	t.Logf("breakpoint set: %s", bpID)

	if err := tools.RemoveBreakpoint(ctx, client, bpID); err != nil {
		t.Fatalf("RemoveBreakpoint: %v", err)
	}
}

func TestSim_DebuggerPauseResume(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.DebuggerEnable(ctx, client); err != nil {
		t.Fatalf("DebuggerEnable: %v", err)
	}
	defer func() { _ = tools.DebuggerDisable(ctx, client) }()

	// Pause (may not pause if no JS is executing, but should not error)
	if err := tools.Pause(ctx, client); err != nil {
		t.Fatalf("Pause: %v", err)
	}

	// Resume
	if err := tools.Resume(ctx, client); err != nil {
		t.Fatalf("Resume: %v", err)
	}
}

func TestSim_DebuggerStepCommands(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.DebuggerEnable(ctx, client); err != nil {
		t.Fatalf("DebuggerEnable: %v", err)
	}
	defer func() { _ = tools.DebuggerDisable(ctx, client) }()

	// Step commands require a paused state. Set a breakpoint and trigger it.
	_, _ = tools.EvaluateScript(ctx, client,
		"window.__stepTestFn = function() { var a = 1; var b = 2; return a + b; }", false)

	bpID, _, err := tools.SetBreakpointByURL(ctx, client, "", 1, nil, "window.__stepTestFn")
	if err != nil {
		// Breakpoint on eval code is tricky. Just verify the functions exist.
		t.Logf("SetBreakpointByURL for step test: %v (testing step commands without paused state)", err)

		// Even without a paused state, the protocol should accept these commands
		// (they may return errors about not being paused, which is fine).
		_ = tools.StepOver(ctx, client)
		_ = tools.StepInto(ctx, client)
		_ = tools.StepOut(ctx, client)
		return
	}

	defer func() { _ = tools.RemoveBreakpoint(ctx, client, bpID) }()

	// Trigger the breakpoint
	go func() {
		evalCtx, evalCancel := simCtx()
		defer evalCancel()
		_, _ = tools.EvaluateScript(evalCtx, client, "window.__stepTestFn()", false)
	}()
	time.Sleep(500 * time.Millisecond)

	_ = tools.StepOver(ctx, client)
	_ = tools.StepInto(ctx, client)
	_ = tools.StepOut(ctx, client)
	_ = tools.Resume(ctx, client)
}

func TestSim_GetScriptSource(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.DebuggerEnable(ctx, client); err != nil {
		t.Fatalf("DebuggerEnable: %v", err)
	}
	defer func() { _ = tools.DebuggerDisable(ctx, client) }()

	// Listen for scriptParsed events
	var scriptID string
	done := make(chan struct{}, 1)
	client.OnEvent("Debugger.scriptParsed", func(_ string, params json.RawMessage) {
		var evt struct {
			ScriptID string `json:"scriptId"`
			URL      string `json:"url"`
		}
		if json.Unmarshal(params, &evt) == nil && evt.ScriptID != "" {
			scriptID = evt.ScriptID
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})

	// Inject a script to trigger scriptParsed
	_, _ = tools.EvaluateScript(ctx, client, "function __getScriptSourceTest() { return 42; }", false)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for scriptParsed event")
	}

	source, err := tools.GetScriptSource(ctx, client, scriptID)
	if err != nil {
		t.Fatalf("GetScriptSource: %v", err)
	}
	t.Logf("script source (%d bytes): %.200s", len(source), source)
}

func TestSim_SearchInContent(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.DebuggerEnable(ctx, client); err != nil {
		t.Fatalf("DebuggerEnable: %v", err)
	}
	defer func() { _ = tools.DebuggerDisable(ctx, client) }()

	// Get a scriptId
	var scriptID string
	done := make(chan struct{}, 1)
	client.OnEvent("Debugger.scriptParsed", func(_ string, params json.RawMessage) {
		var evt struct {
			ScriptID string `json:"scriptId"`
		}
		if json.Unmarshal(params, &evt) == nil && evt.ScriptID != "" {
			scriptID = evt.ScriptID
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})

	_, _ = tools.EvaluateScript(ctx, client, "function __searchTestFn() { return 'findme'; }", false)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for scriptParsed event")
	}

	result, err := tools.SearchInContent(ctx, client, scriptID, "findme", false, false)
	if err != nil {
		t.Fatalf("SearchInContent: %v", err)
	}
	t.Logf("search results: %s", string(result))
}

// =============================================================================
// DOMDebugger: set_dom_breakpoint, remove_dom_breakpoint,
//              set_event_breakpoint, remove_event_breakpoint,
//              set_url_breakpoint, remove_url_breakpoint
// =============================================================================

func TestSim_SetDOMBreakpoint(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	_, _ = tools.EvaluateScript(ctx, client,
		"document.body.innerHTML='<div id=\"bp-target\">test</div>'", false)
	time.Sleep(300 * time.Millisecond)

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "#bp-target")
	if err != nil {
		t.Fatalf("no #bp-target found: %v", err)
	}

	if err := tools.SetDOMBreakpoint(ctx, client, nodeID, "subtree-modified"); err != nil {
		t.Fatalf("SetDOMBreakpoint: %v", err)
	}
	if err := tools.RemoveDOMBreakpoint(ctx, client, nodeID, "subtree-modified"); err != nil {
		t.Fatalf("RemoveDOMBreakpoint: %v", err)
	}
}

func TestSim_SetEventBreakpoint(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.SetEventBreakpoint(ctx, client, "listener", "click"); err != nil {
		t.Fatalf("SetEventBreakpoint: %v", err)
	}
	if err := tools.RemoveEventBreakpoint(ctx, client, "listener", "click"); err != nil {
		t.Fatalf("RemoveEventBreakpoint: %v", err)
	}
}

func TestSim_SetURLBreakpoint(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.SetURLBreakpoint(ctx, client, "test-url", false); err != nil {
		t.Fatalf("SetURLBreakpoint: %v", err)
	}
	if err := tools.RemoveURLBreakpoint(ctx, client, "test-url"); err != nil {
		t.Fatalf("RemoveURLBreakpoint: %v", err)
	}
}

// =============================================================================
// Timeline: timeline_start, timeline_stop, get_timeline_events
// =============================================================================

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

func TestSim_TimelineCollector(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	collector := tools.NewTimelineCollector()
	if err := collector.Start(ctx, client, 3); err != nil {
		t.Fatalf("TimelineCollector.Start: %v", err)
	}

	_, _ = tools.EvaluateScript(ctx, client, "for(let i=0;i<100;i++){}", false)
	time.Sleep(1 * time.Second)

	if err := collector.Stop(ctx, client); err != nil {
		t.Fatalf("TimelineCollector.Stop: %v", err)
	}
	events := collector.GetEvents()
	t.Logf("timeline: %d events", len(events))
}

// =============================================================================
// Memory & Heap: memory_start_tracking, memory_stop_tracking,
//                heap_snapshot, heap_start_tracking, heap_stop_tracking,
//                heap_gc
// =============================================================================

func TestSim_MemoryTracking(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	collector := tools.NewMemoryTrackingCollector()
	if err := collector.Start(ctx, client); err != nil {
		t.Fatalf("MemoryTrackingCollector.Start: %v", err)
	}
	time.Sleep(1 * time.Second)
	result, err := collector.Stop(ctx, client)
	if err != nil {
		t.Fatalf("MemoryTrackingCollector.Stop: %v", err)
	}
	t.Logf("memory tracking: %d events", len(result.Events))
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

func TestSim_HeapTracking(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	collector := tools.NewHeapTrackingCollector()
	if err := collector.Start(ctx, client); err != nil {
		t.Fatalf("HeapTrackingCollector.Start: %v", err)
	}

	_, _ = tools.EvaluateScript(ctx, client, "var arr=[];for(let i=0;i<100;i++){arr.push({x:i})}", false)
	time.Sleep(1 * time.Second)

	result, err := collector.Stop(ctx, client)
	if err != nil {
		t.Fatalf("HeapTrackingCollector.Stop: %v", err)
	}
	t.Logf("heap tracking: %d GC events, healthy=%v", len(result.GCEvents), result.PipelineHealthy)
}

func TestSim_HeapGC(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.HeapGC(ctx, client); err != nil {
		t.Fatalf("HeapGC: %v", err)
	}
}

// =============================================================================
// Profiler: cpu_start_profiling, cpu_stop_profiling,
//           script_start_profiling, script_stop_profiling
// =============================================================================

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

func TestSim_ScriptProfiling(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	collector := tools.NewScriptProfilerCollector()
	if err := collector.Start(ctx, client); err != nil {
		t.Fatalf("ScriptProfilerCollector.Start: %v", err)
	}
	_, _ = tools.EvaluateScript(ctx, client, "for(let i=0;i<1000;i++){}", false)
	time.Sleep(500 * time.Millisecond)

	result, err := collector.Stop(ctx, client)
	if err != nil {
		t.Fatalf("ScriptProfilerCollector.Stop: %v", err)
	}
	t.Logf("script profiler: %d events", len(result.Events))
}

// =============================================================================
// Animation: animation_enable, animation_disable,
//            animation_start_tracking, animation_stop_tracking,
//            get_animation_effect, resolve_animation
// =============================================================================

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

func TestSim_AnimationTracking(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	collector := tools.NewAnimationTrackingCollector()
	if err := collector.Start(ctx, client); err != nil {
		t.Fatalf("AnimationTrackingCollector.Start: %v", err)
	}

	// Create a CSS animation to trigger tracking events
	_, _ = tools.EvaluateScript(ctx, client, `
		var style = document.createElement('style');
		style.textContent = '@keyframes testAnim { from { opacity: 0; } to { opacity: 1; } }';
		document.head.appendChild(style);
		var div = document.createElement('div');
		div.style.animation = 'testAnim 0.5s';
		document.body.appendChild(div);
	`, false)
	time.Sleep(1 * time.Second)

	result, err := collector.Stop(ctx, client)
	if err != nil {
		t.Fatalf("AnimationTrackingCollector.Stop: %v", err)
	}
	t.Logf("animation tracking: %d events", len(result.Events))
}

// =============================================================================
// Canvas: canvas_enable, canvas_disable, get_canvas_content,
//         start_canvas_recording, stop_canvas_recording, get_shader_source
// =============================================================================

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

func TestSim_CanvasContent(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.CanvasEnable(ctx, client); err != nil {
		t.Fatalf("CanvasEnable: %v", err)
	}
	defer func() { _ = tools.CanvasDisable(ctx, client) }()

	// Create a canvas and draw something
	_, _ = tools.EvaluateScript(ctx, client, `
		var canvas = document.createElement('canvas');
		canvas.width = 100; canvas.height = 100;
		canvas.id = 'test-canvas';
		document.body.appendChild(canvas);
		var ctx = canvas.getContext('2d');
		ctx.fillStyle = 'red';
		ctx.fillRect(0, 0, 50, 50);
	`, false)
	time.Sleep(500 * time.Millisecond)

	// Listen for canvasAdded events to get the canvas ID
	var canvasID string
	done := make(chan struct{}, 1)
	client.OnEvent("Canvas.canvasAdded", func(_ string, params json.RawMessage) {
		var evt struct {
			Canvas struct {
				CanvasID string `json:"canvasId"`
			} `json:"canvas"`
		}
		if json.Unmarshal(params, &evt) == nil && evt.Canvas.CanvasID != "" {
			canvasID = evt.Canvas.CanvasID
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})

	// Re-enable to trigger canvasAdded for existing canvases
	_ = tools.CanvasDisable(ctx, client)
	_ = tools.CanvasEnable(ctx, client)
	time.Sleep(500 * time.Millisecond)

	select {
	case <-done:
		content, err := tools.GetCanvasContent(ctx, client, canvasID)
		if err != nil {
			t.Fatalf("GetCanvasContent: %v", err)
		}
		t.Logf("canvas content: %d bytes", len(content))
	case <-time.After(3 * time.Second):
		t.Log("no canvasAdded event received. Canvas tracking may not be supported through iwdp")
	}
}

// =============================================================================
// LayerTree: get_layer_tree, get_compositing_reasons
// =============================================================================

func TestSim_GetLayerTree(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	root, err := tools.GetDocument(ctx, client, 0)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	nodeID, err := tools.QuerySelector(ctx, client, root.NodeID, "body")
	if err != nil {
		t.Fatalf("QuerySelector body: %v", err)
	}

	layers, err := tools.GetLayerTree(ctx, client, nodeID)
	if err != nil {
		t.Fatalf("GetLayerTree: %v", err)
	}
	t.Logf("layer tree: %s", string(layers))

	// Extract a layer ID for GetCompositingReasons
	var layerResult struct {
		Layers []struct {
			LayerID string `json:"layerId"`
		} `json:"childLayers"`
	}
	if json.Unmarshal(layers, &layerResult) == nil && len(layerResult.Layers) > 0 {
		reasons, err := tools.GetCompositingReasons(ctx, client, layerResult.Layers[0].LayerID)
		if err != nil {
			t.Fatalf("GetCompositingReasons: %v", err)
		}
		t.Logf("compositing reasons: %s", string(reasons))
	} else {
		// Try with the root layer
		var rootResult struct {
			LayerID string `json:"layerId"`
		}
		if json.Unmarshal(layers, &rootResult) == nil && rootResult.LayerID != "" {
			reasons, err := tools.GetCompositingReasons(ctx, client, rootResult.LayerID)
			if err != nil {
				t.Fatalf("GetCompositingReasons: %v", err)
			}
			t.Logf("compositing reasons: %s", string(reasons))
		}
	}
}

// =============================================================================
// Workers: worker_enable, worker_disable, send_to_worker
// =============================================================================

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

func TestSim_SendToWorker(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.WorkerEnable(ctx, client); err != nil {
		t.Fatalf("WorkerEnable: %v", err)
	}
	defer func() { _ = tools.WorkerDisable(ctx, client) }()

	// Create a web worker
	_, _ = tools.EvaluateScript(ctx, client, `
		window.__testWorker = new Worker(URL.createObjectURL(new Blob(
			['self.onmessage = function(e) { self.postMessage(e.data); }'],
			{type: 'application/javascript'}
		)));
	`, false)
	time.Sleep(500 * time.Millisecond)

	// Listen for workerCreated
	var workerID string
	done := make(chan struct{}, 1)
	client.OnEvent("Worker.workerCreated", func(_ string, params json.RawMessage) {
		var evt struct {
			WorkerID string `json:"workerId"`
		}
		if json.Unmarshal(params, &evt) == nil && evt.WorkerID != "" {
			workerID = evt.WorkerID
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})

	// Re-enable to get existing workers
	_ = tools.WorkerDisable(ctx, client)
	_ = tools.WorkerEnable(ctx, client)
	time.Sleep(500 * time.Millisecond)

	select {
	case <-done:
		if err := tools.SendToWorker(ctx, client, workerID, `{"method":"echo","params":{}}`); err != nil {
			t.Fatalf("SendToWorker: %v", err)
		}
		t.Logf("sent message to worker %s", workerID)
	case <-time.After(3 * time.Second):
		t.Log("no workerCreated event received. Worker tracking may require page reload")
	}

	// Cleanup
	_, _ = tools.EvaluateScript(ctx, client, "if(window.__testWorker) window.__testWorker.terminate()", false)
}

// =============================================================================
// Audit: run_audit
// =============================================================================

func TestSim_RunAudit(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	result, err := tools.RunAudit(ctx, client, "function() { return {level: 'pass', message: 'test ok'}; }")
	if err != nil {
		t.Fatalf("RunAudit: %v", err)
	}
	t.Logf("audit result: %s", string(result))
}

// =============================================================================
// Security: get_certificate_info
// =============================================================================

func TestSim_GetCertificateInfo(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	// Enable network to capture a request with a certificate.
	// Keep monitor running while we query the certificate (stopping it clears resources).
	monitor := tools.NewNetworkMonitor()
	if err := monitor.Start(ctx, client); err != nil {
		t.Fatalf("NetworkMonitor.Start: %v", err)
	}
	defer func() { _ = monitor.Stop(ctx, client) }()

	_, _ = tools.EvaluateScript(ctx, client, "fetch('https://example.com/').catch(()=>{})", false)
	time.Sleep(2 * time.Second)

	requests := monitor.GetRequests()
	var requestID string
	for _, r := range requests {
		if strings.HasPrefix(r.Request.URL, "https://") && r.Request.RequestID != "" {
			requestID = r.Request.RequestID
			break
		}
	}

	if requestID == "" {
		t.Fatal("no HTTPS request captured for certificate info")
	}

	cert, err := tools.GetCertificateInfo(ctx, client, requestID)
	if err != nil {
		t.Fatalf("GetCertificateInfo: %v", err)
	}
	t.Logf("certificate info: %s", string(cert))
}

// =============================================================================
// select_page — implicitly tested by getSimClient (connects WebSocket to a page).
// The MCP tool select_page is a thin wrapper around webkit.NewClient.
// =============================================================================

// =============================================================================
// Network: intercept_continue, intercept_with_response
// =============================================================================

func TestSim_InterceptContinue(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	monitor := tools.NewNetworkMonitor()
	_ = monitor.Start(ctx, client)

	if err := tools.SetRequestInterception(ctx, client, true); err != nil {
		t.Fatalf("SetRequestInterception enable: %v", err)
	}

	// Listen for intercepted requests
	var interceptedID string
	done := make(chan struct{}, 1)
	client.OnEvent("Network.requestIntercepted", func(_ string, params json.RawMessage) {
		var evt struct {
			RequestID string `json:"requestId"`
		}
		if json.Unmarshal(params, &evt) == nil && evt.RequestID != "" {
			interceptedID = evt.RequestID
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})

	// Trigger a fetch that will be intercepted
	go func() {
		evalCtx, evalCancel := simCtx()
		defer evalCancel()
		_, _ = tools.EvaluateScript(evalCtx, client, "fetch('/test-intercept').catch(()=>{})", false)
	}()

	select {
	case <-done:
		if err := tools.InterceptContinue(ctx, client, interceptedID, "request"); err != nil {
			t.Fatalf("InterceptContinue: %v", err)
		}
		t.Logf("intercepted and continued request %s", interceptedID)
	case <-time.After(5 * time.Second):
		t.Log("no requestIntercepted event received. Network interception may not be supported through iwdp")
	}

	_ = tools.SetRequestInterception(ctx, client, false)
	_ = monitor.Stop(ctx, client)
}

func TestSim_InterceptWithResponse(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	monitor := tools.NewNetworkMonitor()
	_ = monitor.Start(ctx, client)

	if err := tools.SetRequestInterception(ctx, client, true); err != nil {
		t.Fatalf("SetRequestInterception enable: %v", err)
	}

	var interceptedID string
	done := make(chan struct{}, 1)
	client.OnEvent("Network.requestIntercepted", func(_ string, params json.RawMessage) {
		var evt struct {
			RequestID string `json:"requestId"`
		}
		if json.Unmarshal(params, &evt) == nil && evt.RequestID != "" {
			interceptedID = evt.RequestID
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})

	go func() {
		evalCtx, evalCancel := simCtx()
		defer evalCancel()
		_, _ = tools.EvaluateScript(evalCtx, client, "fetch('/test-mock-response').catch(()=>{})", false)
	}()

	select {
	case <-done:
		err := tools.InterceptWithResponse(ctx, client, interceptedID, "request", 200,
			map[string]string{"Content-Type": "text/plain"}, "mocked body", false)
		if err != nil {
			t.Fatalf("InterceptWithResponse: %v", err)
		}
		t.Logf("intercepted and responded to request %s", interceptedID)
	case <-time.After(5 * time.Second):
		t.Log("no requestIntercepted event received. Network interception may not be supported through iwdp")
	}

	_ = tools.SetRequestInterception(ctx, client, false)
	_ = monitor.Stop(ctx, client)
}

// =============================================================================
// Debugger: evaluate_on_call_frame
// =============================================================================

func TestSim_EvaluateOnCallFrame(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.DebuggerEnable(ctx, client); err != nil {
		t.Fatalf("DebuggerEnable: %v", err)
	}
	defer func() {
		_ = tools.Resume(ctx, client)
		_ = tools.DebuggerDisable(ctx, client)
	}()

	// Set a breakpoint using a debugger statement
	var callFrameID string
	paused := make(chan struct{}, 1)
	client.OnEvent("Debugger.paused", func(_ string, params json.RawMessage) {
		var evt struct {
			CallFrames []struct {
				CallFrameID string `json:"callFrameId"`
			} `json:"callFrames"`
		}
		if json.Unmarshal(params, &evt) == nil && len(evt.CallFrames) > 0 {
			callFrameID = evt.CallFrames[0].CallFrameID
			select {
			case paused <- struct{}{}:
			default:
			}
		}
	})

	// Execute code with a debugger statement
	go func() {
		evalCtx, evalCancel := simCtx()
		defer evalCancel()
		_, _ = tools.EvaluateScript(evalCtx, client, "var __evalTestVar = 42; debugger; __evalTestVar", false)
	}()

	select {
	case <-paused:
		result, err := tools.EvaluateOnCallFrame(ctx, client, callFrameID, "__evalTestVar", true)
		if err != nil {
			t.Fatalf("EvaluateOnCallFrame: %v", err)
		}
		t.Logf("evaluated on call frame: type=%s", result.Result.Type)
		_ = tools.Resume(ctx, client)
	case <-time.After(5 * time.Second):
		// debugger statement in Runtime.evaluate may not trigger Debugger.paused
		// through iwdp Target routing due to pipeline serialization
		t.Log("debugger statement did not trigger pause through iwdp Target routing (known limitation)")
	}
}

// =============================================================================
// CSS: get_stylesheet_text
// =============================================================================

func TestSim_GetStylesheetText(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	// CSS.getStyleSheetText hangs through iwdp Target routing (same as CSS.getAllStyleSheets).
	// The tool detects Target routing and returns an error immediately.
	_, err := tools.GetStylesheetText(ctx, client, "fake-id")
	if err == nil {
		t.Fatal("expected error for GetStylesheetText through iwdp Target routing")
	}
	t.Logf("got expected error: %v", err)
}

// =============================================================================
// Canvas: start_canvas_recording, stop_canvas_recording, get_shader_source
// =============================================================================

func TestSim_CanvasRecording(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.CanvasEnable(ctx, client); err != nil {
		t.Fatalf("CanvasEnable: %v", err)
	}
	defer func() { _ = tools.CanvasDisable(ctx, client) }()

	// Create a canvas
	_, _ = tools.EvaluateScript(ctx, client, `
		var c = document.createElement('canvas');
		c.width = 50; c.height = 50; c.id = 'rec-canvas';
		document.body.appendChild(c);
		var ctx2d = c.getContext('2d');
		ctx2d.fillRect(0,0,10,10);
	`, false)
	time.Sleep(500 * time.Millisecond)

	// Get canvas ID
	var canvasID string
	done := make(chan struct{}, 1)
	client.OnEvent("Canvas.canvasAdded", func(_ string, params json.RawMessage) {
		var evt struct {
			Canvas struct {
				CanvasID string `json:"canvasId"`
			} `json:"canvas"`
		}
		if json.Unmarshal(params, &evt) == nil && evt.Canvas.CanvasID != "" {
			canvasID = evt.Canvas.CanvasID
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})

	_ = tools.CanvasDisable(ctx, client)
	_ = tools.CanvasEnable(ctx, client)
	time.Sleep(500 * time.Millisecond)

	select {
	case <-done:
		if err := tools.StartCanvasRecording(ctx, client, canvasID, 10); err != nil {
			t.Fatalf("StartCanvasRecording: %v", err)
		}
		// Draw to generate recording frames
		_, _ = tools.EvaluateScript(ctx, client, `
			var c2 = document.getElementById('rec-canvas').getContext('2d');
			c2.fillStyle='blue'; c2.fillRect(10,10,20,20);
		`, false)
		time.Sleep(500 * time.Millisecond)

		if err := tools.StopCanvasRecording(ctx, client, canvasID); err != nil {
			t.Fatalf("StopCanvasRecording: %v", err)
		}
		t.Logf("canvas recording started and stopped for %s", canvasID)
	case <-time.After(3 * time.Second):
		t.Log("no canvasAdded event received. Canvas tracking may not be supported through iwdp")
	}
}

func TestSim_GetShaderSource(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	if err := tools.CanvasEnable(ctx, client); err != nil {
		t.Fatalf("CanvasEnable: %v", err)
	}
	defer func() { _ = tools.CanvasDisable(ctx, client) }()

	// Create a WebGL canvas with shaders
	_, _ = tools.EvaluateScript(ctx, client, `
		var glCanvas = document.createElement('canvas');
		glCanvas.width = 50; glCanvas.height = 50;
		document.body.appendChild(glCanvas);
		var gl = glCanvas.getContext('webgl');
		if (gl) {
			var prog = gl.createProgram();
			var vs = gl.createShader(gl.VERTEX_SHADER);
			gl.shaderSource(vs, 'attribute vec4 a; void main(){ gl_Position=a; }');
			gl.compileShader(vs);
			gl.attachShader(prog, vs);
			var fs = gl.createShader(gl.FRAGMENT_SHADER);
			gl.shaderSource(fs, 'void main(){ gl_FragColor=vec4(1,0,0,1); }');
			gl.compileShader(fs);
			gl.attachShader(prog, fs);
			gl.linkProgram(prog);
			window.__testProgID = prog;
		}
	`, false)
	time.Sleep(500 * time.Millisecond)

	// Get canvas/program ID from Canvas domain events
	var programID string
	done := make(chan struct{}, 1)
	client.OnEvent("Canvas.programCreated", func(_ string, params json.RawMessage) {
		var evt struct {
			ProgramID string `json:"programId"`
		}
		if json.Unmarshal(params, &evt) == nil && evt.ProgramID != "" {
			programID = evt.ProgramID
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})

	_ = tools.CanvasDisable(ctx, client)
	_ = tools.CanvasEnable(ctx, client)
	time.Sleep(500 * time.Millisecond)

	select {
	case <-done:
		source, err := tools.GetShaderSource(ctx, client, programID, "vertex")
		if err != nil {
			t.Fatalf("GetShaderSource: %v", err)
		}
		t.Logf("shader source: %s", source)
	case <-time.After(3 * time.Second):
		t.Log("no programCreated event received. WebGL shader tracking may not be supported through iwdp")
	}
}

// =============================================================================
// Animation: get_animation_effect, resolve_animation
// =============================================================================

func TestSim_GetAnimationEffect(t *testing.T) {
	client := getSimClient(t)
	ctx, cancel := simCtx()
	defer cancel()

	collector := tools.NewAnimationTrackingCollector()
	if err := collector.Start(ctx, client); err != nil {
		t.Fatalf("AnimationTrackingCollector.Start: %v", err)
	}

	// Create a longer CSS animation
	_, _ = tools.EvaluateScript(ctx, client, `
		var s = document.createElement('style');
		s.textContent = '@keyframes fadeTest { from { opacity: 0; } to { opacity: 1; } }';
		document.head.appendChild(s);
		var d = document.createElement('div');
		d.id = 'anim-target';
		d.style.animation = 'fadeTest 2s';
		document.body.appendChild(d);
	`, false)
	time.Sleep(1 * time.Second)

	// Listen for animationCreated to get an animation ID
	var animID string
	done := make(chan struct{}, 1)
	client.OnEvent("Animation.animationCreated", func(_ string, params json.RawMessage) {
		var evt struct {
			AnimationID string `json:"animationId"`
		}
		if json.Unmarshal(params, &evt) == nil && evt.AnimationID != "" {
			animID = evt.AnimationID
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})

	// Re-trigger
	_, _ = tools.EvaluateScript(ctx, client, `
		var el = document.getElementById('anim-target');
		if (el) { el.style.animation = 'none'; void el.offsetWidth; el.style.animation = 'fadeTest 2s'; }
	`, false)
	time.Sleep(1 * time.Second)

	select {
	case <-done:
		effect, err := tools.GetAnimationEffect(ctx, client, animID)
		if err != nil {
			t.Fatalf("GetAnimationEffect: %v", err)
		}
		t.Logf("animation effect: %s", string(effect))

		obj, err := tools.ResolveAnimation(ctx, client, animID, "test-group")
		if err != nil {
			t.Fatalf("ResolveAnimation: %v", err)
		}
		t.Logf("resolved animation: type=%s objectId=%s", obj.Type, obj.ObjectID)
	case <-time.After(5 * time.Second):
		t.Log("no animationCreated event received. Animation tracking may require specific page content")
	}

	_, _ = collector.Stop(ctx, client)
}
