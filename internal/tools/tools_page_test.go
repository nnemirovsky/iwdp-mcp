package tools_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nnemirovsky/iwdp-mcp/internal/tools"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit/testutil"
)

// Ensure imports are used.
var (
	_ *testutil.MockServer
	_ webkit.Cookie
)

// ---------------------------------------------------------------------------
// Page (page.go)
// ---------------------------------------------------------------------------

func TestReload(t *testing.T) {
	mock, client := setup(t)

	var receivedIgnoreCache bool
	mock.Handle("Page.reload", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			IgnoreCache bool `json:"ignoreCache"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedIgnoreCache = p.IgnoreCache
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.Reload(ctx, client, true); err != nil {
		t.Fatalf("Reload returned error: %v", err)
	}
	if !receivedIgnoreCache {
		t.Errorf("expected ignoreCache=true, got false")
	}
}

func TestTakeScreenshot(t *testing.T) {
	mock, client := setup(t)

	mock.HandleFunc("Runtime.evaluate", map[string]interface{}{
		"result": map[string]interface{}{
			"type":  "string",
			"value": `{"width":1024,"height":768}`,
		},
	})
	mock.HandleFunc("Page.snapshotRect", map[string]interface{}{
		"dataURL": "data:image/png;base64,abc123",
	})

	ctx := context.Background()
	dataURL, err := tools.TakeScreenshot(ctx, client)
	if err != nil {
		t.Fatalf("TakeScreenshot returned error: %v", err)
	}
	if dataURL != "data:image/png;base64,abc123" {
		t.Errorf("expected dataURL %q, got %q", "data:image/png;base64,abc123", dataURL)
	}
}

func TestSnapshotNode(t *testing.T) {
	mock, client := setup(t)

	var receivedNodeID int
	mock.Handle("Page.snapshotNode", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			NodeID int `json:"nodeId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedNodeID = p.NodeID
		return map[string]interface{}{
			"dataURL": "data:image/png;base64,node123",
		}, nil
	})

	ctx := context.Background()
	dataURL, err := tools.SnapshotNode(ctx, client, 42)
	if err != nil {
		t.Fatalf("SnapshotNode returned error: %v", err)
	}
	if receivedNodeID != 42 {
		t.Errorf("expected nodeId=42, got %d", receivedNodeID)
	}
	if dataURL != "data:image/png;base64,node123" {
		t.Errorf("expected dataURL %q, got %q", "data:image/png;base64,node123", dataURL)
	}
}

func TestSetCookie(t *testing.T) {
	mock, client := setup(t)

	var receivedName, receivedDomain string
	mock.Handle("Page.setCookie", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			Cookie webkit.Cookie `json:"cookie"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedName = p.Cookie.Name
		receivedDomain = p.Cookie.Domain
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	cookie := webkit.Cookie{
		Name:   "session",
		Value:  "xyz",
		Domain: ".example.com",
		Path:   "/",
	}
	if err := tools.SetCookie(ctx, client, cookie); err != nil {
		t.Fatalf("SetCookie returned error: %v", err)
	}
	if receivedName != "session" {
		t.Errorf("expected cookie name %q, got %q", "session", receivedName)
	}
	if receivedDomain != ".example.com" {
		t.Errorf("expected cookie domain %q, got %q", ".example.com", receivedDomain)
	}
}

func TestDeleteCookie(t *testing.T) {
	mock, client := setup(t)

	var receivedCookieName, receivedURL string
	mock.Handle("Page.deleteCookie", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			CookieName string `json:"cookieName"`
			URL        string `json:"url"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedCookieName = p.CookieName
		receivedURL = p.URL
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.DeleteCookie(ctx, client, "session", "https://example.com"); err != nil {
		t.Fatalf("DeleteCookie returned error: %v", err)
	}
	if receivedCookieName != "session" {
		t.Errorf("expected cookieName %q, got %q", "session", receivedCookieName)
	}
	if receivedURL != "https://example.com" {
		t.Errorf("expected url %q, got %q", "https://example.com", receivedURL)
	}
}

// ---------------------------------------------------------------------------
// Runtime (runtime.go)
// ---------------------------------------------------------------------------

func TestEvaluateScriptWasThrown(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("Runtime.evaluate", map[string]interface{}{
		"result": map[string]interface{}{
			"type": "object",
		},
		"wasThrown": true,
		"exceptionDetails": map[string]interface{}{
			"text": "ReferenceError: foo is not defined",
			"line": 1,
			"exception": map[string]interface{}{
				"type":        "object",
				"description": "ReferenceError: foo is not defined",
			},
		},
	})

	ctx := context.Background()
	result, err := tools.EvaluateScript(ctx, client, "foo", true)
	if err == nil {
		t.Fatal("expected error for wasThrown=true, got nil")
	}
	if result == nil {
		t.Fatal("expected non-nil result even when wasThrown=true")
	}
	if !result.WasThrown {
		t.Errorf("expected WasThrown=true")
	}
}

func TestCallFunctionOn(t *testing.T) {
	mock, client := setup(t)

	var receivedObjectID, receivedFunc string
	mock.Handle("Runtime.callFunctionOn", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			ObjectID            string `json:"objectId"`
			FunctionDeclaration string `json:"functionDeclaration"`
			ReturnByValue       bool   `json:"returnByValue"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedObjectID = p.ObjectID
		receivedFunc = p.FunctionDeclaration
		return map[string]interface{}{
			"result": map[string]interface{}{
				"type":  "string",
				"value": `"result"`,
			},
			"wasThrown": false,
		}, nil
	})

	ctx := context.Background()
	result, err := tools.CallFunctionOn(ctx, client, "obj1", "function(){}", nil, true)
	if err != nil {
		t.Fatalf("CallFunctionOn returned error: %v", err)
	}
	if receivedObjectID != "obj1" {
		t.Errorf("expected objectId %q, got %q", "obj1", receivedObjectID)
	}
	if receivedFunc != "function(){}" {
		t.Errorf("expected functionDeclaration %q, got %q", "function(){}", receivedFunc)
	}
	if result.Result.Type != "string" {
		t.Errorf("expected result type %q, got %q", "string", result.Result.Type)
	}
}

func TestCallFunctionOnWithArgs(t *testing.T) {
	mock, client := setup(t)

	var receivedArgs []map[string]interface{}
	mock.Handle("Runtime.callFunctionOn", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			Arguments []map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedArgs = p.Arguments
		return map[string]interface{}{
			"result": map[string]interface{}{
				"type": "undefined",
			},
			"wasThrown": false,
		}, nil
	})

	ctx := context.Background()
	args := []interface{}{"a", 1}
	_, err := tools.CallFunctionOn(ctx, client, "obj1", "function(){}", args, true)
	if err != nil {
		t.Fatalf("CallFunctionOnWithArgs returned error: %v", err)
	}
	if len(receivedArgs) != 2 {
		t.Fatalf("expected 2 arguments, got %d", len(receivedArgs))
	}
	if receivedArgs[0]["value"] != "a" {
		t.Errorf("expected first arg value %q, got %v", "a", receivedArgs[0]["value"])
	}
	// JSON numbers unmarshal as float64.
	if receivedArgs[1]["value"] != float64(1) {
		t.Errorf("expected second arg value 1, got %v", receivedArgs[1]["value"])
	}
}

func TestGetProperties(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("Runtime.getProperties", map[string]interface{}{
		"properties": []map[string]interface{}{
			{
				"name":         "length",
				"configurable": true,
				"enumerable":   true,
				"value": map[string]interface{}{
					"type":  "number",
					"value": 5,
				},
			},
			{
				"name":         "name",
				"configurable": true,
				"enumerable":   true,
				"value": map[string]interface{}{
					"type":  "string",
					"value": `"test"`,
				},
			},
		},
	})

	ctx := context.Background()
	props, err := tools.GetProperties(ctx, client, "obj1", true)
	if err != nil {
		t.Fatalf("GetProperties returned error: %v", err)
	}
	if len(props) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(props))
	}
	if props[0].Name != "length" {
		t.Errorf("expected first property name %q, got %q", "length", props[0].Name)
	}
	if props[1].Name != "name" {
		t.Errorf("expected second property name %q, got %q", "name", props[1].Name)
	}
}

// ---------------------------------------------------------------------------
// DOM (dom.go)
// ---------------------------------------------------------------------------

func TestQuerySelectorAll(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("DOM.querySelectorAll", map[string]interface{}{
		"nodeIds": []int{10, 20, 30},
	})

	ctx := context.Background()
	nodeIDs, err := tools.QuerySelectorAll(ctx, client, 1, "div")
	if err != nil {
		t.Fatalf("QuerySelectorAll returned error: %v", err)
	}
	if len(nodeIDs) != 3 {
		t.Fatalf("expected 3 nodeIds, got %d", len(nodeIDs))
	}
	expected := []int{10, 20, 30}
	for i, id := range nodeIDs {
		if id != expected[i] {
			t.Errorf("nodeIDs[%d]: expected %d, got %d", i, expected[i], id)
		}
	}
}

func TestGetAttributes(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("DOM.getAttributes", map[string]interface{}{
		"attributes": []string{"class", "main", "id", "content"},
	})

	ctx := context.Background()
	attrs, err := tools.GetAttributes(ctx, client, 42)
	if err != nil {
		t.Fatalf("GetAttributes returned error: %v", err)
	}
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attribute pairs, got %d", len(attrs))
	}
	if attrs["class"] != "main" {
		t.Errorf("expected class=%q, got %q", "main", attrs["class"])
	}
	if attrs["id"] != "content" {
		t.Errorf("expected id=%q, got %q", "content", attrs["id"])
	}
}

func TestGetEventListeners(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("DOM.getEventListenersForNode", map[string]interface{}{
		"listeners": []map[string]interface{}{
			{
				"type":        "click",
				"useCapture":  false,
				"isAttribute": false,
				"nodeId":      42,
				"handlerBody": "function onClick() {}",
			},
			{
				"type":        "keydown",
				"useCapture":  true,
				"isAttribute": true,
				"nodeId":      42,
				"handlerBody": "function onKeyDown() {}",
			},
		},
	})

	ctx := context.Background()
	listeners, err := tools.GetEventListeners(ctx, client, 42)
	if err != nil {
		t.Fatalf("GetEventListeners returned error: %v", err)
	}
	if len(listeners) != 2 {
		t.Fatalf("expected 2 listeners, got %d", len(listeners))
	}
	if listeners[0].Type != "click" {
		t.Errorf("expected first listener type %q, got %q", "click", listeners[0].Type)
	}
	if listeners[0].UseCapture {
		t.Errorf("expected first listener useCapture=false")
	}
	if listeners[1].Type != "keydown" {
		t.Errorf("expected second listener type %q, got %q", "keydown", listeners[1].Type)
	}
	if !listeners[1].UseCapture {
		t.Errorf("expected second listener useCapture=true")
	}
	if !listeners[1].IsAttribute {
		t.Errorf("expected second listener isAttribute=true")
	}
}

func TestHighlightNode(t *testing.T) {
	mock, client := setup(t)

	var receivedNodeID int
	mock.Handle("DOM.highlightNode", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			NodeID int `json:"nodeId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedNodeID = p.NodeID
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.HighlightNode(ctx, client, 42); err != nil {
		t.Fatalf("HighlightNode returned error: %v", err)
	}
	if receivedNodeID != 42 {
		t.Errorf("expected nodeId=42, got %d", receivedNodeID)
	}
}

func TestHideHighlight(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("DOM.hideHighlight", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.HideHighlight(ctx, client); err != nil {
		t.Fatalf("HideHighlight returned error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CSS (css.go)
// ---------------------------------------------------------------------------

func TestGetMatchedStyles(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("CSS.getMatchedStylesForNode", map[string]interface{}{
		"matchedCSSRules": []map[string]interface{}{
			{
				"rule": map[string]interface{}{
					"origin": "regular",
					"style": map[string]interface{}{
						"cssText": "color: red",
					},
				},
			},
		},
	})

	ctx := context.Background()
	result, err := tools.GetMatchedStyles(ctx, client, 42)
	if err != nil {
		t.Fatalf("GetMatchedStyles returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Verify the raw JSON contains the expected content.
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}
	if _, ok := parsed["matchedCSSRules"]; !ok {
		t.Errorf("expected matchedCSSRules key in result")
	}
}

func TestGetComputedStyle(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("CSS.getComputedStyleForNode", map[string]interface{}{
		"computedStyle": []map[string]interface{}{
			{"name": "color", "value": "red"},
		},
	})

	ctx := context.Background()
	style, err := tools.GetComputedStyle(ctx, client, 42)
	if err != nil {
		t.Fatalf("GetComputedStyle returned error: %v", err)
	}
	if len(style) != 1 {
		t.Fatalf("expected 1 computed property, got %d", len(style))
	}
	if style[0].Name != "color" {
		t.Errorf("expected name %q, got %q", "color", style[0].Name)
	}
	if style[0].Value != "red" {
		t.Errorf("expected value %q, got %q", "red", style[0].Value)
	}
}

func TestGetInlineStyles(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("CSS.getInlineStylesForNode", map[string]interface{}{
		"inlineStyle": map[string]interface{}{
			"cssText": "color: red",
		},
	})

	ctx := context.Background()
	style, err := tools.GetInlineStyles(ctx, client, 42)
	if err != nil {
		t.Fatalf("GetInlineStyles returned error: %v", err)
	}
	if style == nil {
		t.Fatal("expected non-nil inline style")
	}
	if style.Text != "color: red" {
		t.Errorf("expected cssText %q, got %q", "color: red", style.Text)
	}
}

func TestSetStyleText(t *testing.T) {
	mock, client := setup(t)

	var receivedText string
	mock.Handle("CSS.setStyleText", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedText = p.Text
		return map[string]interface{}{
			"style": map[string]interface{}{
				"cssText": "color: blue",
			},
		}, nil
	})

	ctx := context.Background()
	styleID := json.RawMessage(`{"styleSheetId":"ss1","ordinal":0}`)
	style, err := tools.SetStyleText(ctx, client, styleID, "color: blue")
	if err != nil {
		t.Fatalf("SetStyleText returned error: %v", err)
	}
	if receivedText != "color: blue" {
		t.Errorf("expected text %q, got %q", "color: blue", receivedText)
	}
	if style == nil {
		t.Fatal("expected non-nil style result")
	}
	if style.Text != "color: blue" {
		t.Errorf("expected style cssText %q, got %q", "color: blue", style.Text)
	}
}

func TestGetAllStylesheets(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("CSS.getAllStyleSheets", map[string]interface{}{
		"headers": []map[string]interface{}{
			{"styleSheetId": "ss1", "sourceURL": "style.css"},
		},
	})

	ctx := context.Background()
	sheets, err := tools.GetAllStylesheets(ctx, client)
	if err != nil {
		t.Fatalf("GetAllStylesheets returned error: %v", err)
	}
	if len(sheets) != 1 {
		t.Fatalf("expected 1 stylesheet, got %d", len(sheets))
	}
	if sheets[0].StyleSheetID != "ss1" {
		t.Errorf("expected styleSheetId %q, got %q", "ss1", sheets[0].StyleSheetID)
	}
	if sheets[0].SourceURL != "style.css" {
		t.Errorf("expected sourceURL %q, got %q", "style.css", sheets[0].SourceURL)
	}
}

func TestGetStylesheetText(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("CSS.getStyleSheetText", map[string]interface{}{
		"text": "body { color: red }",
	})

	ctx := context.Background()
	text, err := tools.GetStylesheetText(ctx, client, "ss1")
	if err != nil {
		t.Fatalf("GetStylesheetText returned error: %v", err)
	}
	if text != "body { color: red }" {
		t.Errorf("expected text %q, got %q", "body { color: red }", text)
	}
}

func TestForcePseudoState(t *testing.T) {
	mock, client := setup(t)

	var receivedNodeID int
	var receivedPseudoClasses []string
	mock.Handle("CSS.forcePseudoState", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			NodeID              int      `json:"nodeId"`
			ForcedPseudoClasses []string `json:"forcedPseudoClasses"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedNodeID = p.NodeID
		receivedPseudoClasses = p.ForcedPseudoClasses
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.ForcePseudoState(ctx, client, 42, []string{"hover", "focus"}); err != nil {
		t.Fatalf("ForcePseudoState returned error: %v", err)
	}
	if receivedNodeID != 42 {
		t.Errorf("expected nodeId=42, got %d", receivedNodeID)
	}
	if len(receivedPseudoClasses) != 2 {
		t.Fatalf("expected 2 pseudo classes, got %d", len(receivedPseudoClasses))
	}
	if receivedPseudoClasses[0] != "hover" {
		t.Errorf("expected first pseudo class %q, got %q", "hover", receivedPseudoClasses[0])
	}
	if receivedPseudoClasses[1] != "focus" {
		t.Errorf("expected second pseudo class %q, got %q", "focus", receivedPseudoClasses[1])
	}
}

// ---------------------------------------------------------------------------
// Interaction (interaction.go)
// ---------------------------------------------------------------------------

func TestTypeText(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("Runtime.evaluate", map[string]interface{}{
		"result": map[string]interface{}{
			"type": "undefined",
		},
		"wasThrown": false,
	})

	ctx := context.Background()
	if err := tools.TypeText(ctx, client, "hello"); err != nil {
		t.Fatalf("TypeText returned error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Storage (storage.go)
// ---------------------------------------------------------------------------

func TestGetLocalStorage(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("DOMStorage.getDOMStorageItems", map[string]interface{}{
		"entries": [][]string{
			{"key1", "val1"},
			{"key2", "val2"},
		},
	})

	ctx := context.Background()
	items, err := tools.GetLocalStorage(ctx, client, "https://example.com")
	if err != nil {
		t.Fatalf("GetLocalStorage returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Key != "key1" || items[0].Value != "val1" {
		t.Errorf("expected first item key1=val1, got %s=%s", items[0].Key, items[0].Value)
	}
	if items[1].Key != "key2" || items[1].Value != "val2" {
		t.Errorf("expected second item key2=val2, got %s=%s", items[1].Key, items[1].Value)
	}
}

func TestSetLocalStorageItem(t *testing.T) {
	mock, client := setup(t)

	var receivedKey, receivedValue string
	mock.Handle("DOMStorage.setDOMStorageItem", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedKey = p.Key
		receivedValue = p.Value
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.SetLocalStorageItem(ctx, client, "https://example.com", "key", "val"); err != nil {
		t.Fatalf("SetLocalStorageItem returned error: %v", err)
	}
	if receivedKey != "key" {
		t.Errorf("expected key %q, got %q", "key", receivedKey)
	}
	if receivedValue != "val" {
		t.Errorf("expected value %q, got %q", "val", receivedValue)
	}
}

func TestRemoveLocalStorageItem(t *testing.T) {
	mock, client := setup(t)

	var receivedKey string
	mock.Handle("DOMStorage.removeDOMStorageItem", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			Key string `json:"key"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedKey = p.Key
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.RemoveLocalStorageItem(ctx, client, "https://example.com", "key"); err != nil {
		t.Fatalf("RemoveLocalStorageItem returned error: %v", err)
	}
	if receivedKey != "key" {
		t.Errorf("expected key %q, got %q", "key", receivedKey)
	}
}

func TestClearLocalStorage(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("DOMStorage.clearDOMStorageItems", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.ClearLocalStorage(ctx, client, "https://example.com"); err != nil {
		t.Fatalf("ClearLocalStorage returned error: %v", err)
	}
}

func TestGetSessionStorage(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("DOMStorage.getDOMStorageItems", map[string]interface{}{
		"entries": [][]string{
			{"skey1", "sval1"},
		},
	})

	ctx := context.Background()
	items, err := tools.GetSessionStorage(ctx, client, "https://example.com")
	if err != nil {
		t.Fatalf("GetSessionStorage returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Key != "skey1" || items[0].Value != "sval1" {
		t.Errorf("expected skey1=sval1, got %s=%s", items[0].Key, items[0].Value)
	}
}

func TestListIndexedDatabases(t *testing.T) {
	mock, client := setup(t)

	mock.HandleFunc("IndexedDB.requestDatabaseNames", map[string]interface{}{
		"databaseNames": []string{"mydb"},
	})
	mock.HandleFunc("IndexedDB.requestDatabase", map[string]interface{}{
		"databaseWithObjectStores": map[string]interface{}{
			"name":    "mydb",
			"version": 1,
			"objectStores": []map[string]interface{}{
				{
					"name":          "store1",
					"keyPath":       "id",
					"autoIncrement": true,
					"indexes":       []interface{}{},
				},
			},
		},
	})

	ctx := context.Background()
	dbs, err := tools.ListIndexedDatabases(ctx, client, "https://example.com")
	if err != nil {
		t.Fatalf("ListIndexedDatabases returned error: %v", err)
	}
	if len(dbs) != 1 {
		t.Fatalf("expected 1 database, got %d", len(dbs))
	}
	if dbs[0].Name != "mydb" {
		t.Errorf("expected db name %q, got %q", "mydb", dbs[0].Name)
	}
	if dbs[0].Version != 1 {
		t.Errorf("expected db version 1, got %d", dbs[0].Version)
	}
	if len(dbs[0].ObjectStores) != 1 {
		t.Fatalf("expected 1 object store, got %d", len(dbs[0].ObjectStores))
	}
	if dbs[0].ObjectStores[0].Name != "store1" {
		t.Errorf("expected object store name %q, got %q", "store1", dbs[0].ObjectStores[0].Name)
	}
}

func TestGetIndexedDBData(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("IndexedDB.requestData", map[string]interface{}{
		"objectStoreDataEntries": []map[string]interface{}{
			{"key": map[string]interface{}{"type": "number", "value": 1}, "primaryKey": map[string]interface{}{"type": "number", "value": 1}, "value": map[string]interface{}{"type": "string", "value": "data1"}},
		},
		"hasMore": false,
	})

	ctx := context.Background()
	result, err := tools.GetIndexedDBData(ctx, client, "https://example.com", "mydb", "store1", 0, 10)
	if err != nil {
		t.Fatalf("GetIndexedDBData returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	if _, ok := parsed["objectStoreDataEntries"]; !ok {
		t.Errorf("expected objectStoreDataEntries in result")
	}
}

func TestClearIndexedDBStore(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("IndexedDB.clearObjectStore", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.ClearIndexedDBStore(ctx, client, "https://example.com", "mydb", "store1"); err != nil {
		t.Fatalf("ClearIndexedDBStore returned error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Console (console.go)
// ---------------------------------------------------------------------------

func TestConsoleCollectorClear(t *testing.T) {
	mock, client := setup(t)

	mock.HandleFunc("Console.enable", map[string]interface{}{})

	collector := tools.NewConsoleCollector()
	ctx := context.Background()
	if err := collector.Start(ctx, client); err != nil {
		t.Fatalf("ConsoleCollector.Start: %v", err)
	}

	// Send a console event.
	if err := mock.SendEvent("Console.messageAdded", map[string]interface{}{
		"message": map[string]interface{}{
			"source": "console-api",
			"level":  "log",
			"text":   "test message",
		},
	}); err != nil {
		t.Fatalf("SendEvent: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	msgs := collector.GetMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message before clear, got %d", len(msgs))
	}

	collector.Clear()

	msgs = collector.GetMessages()
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after clear, got %d", len(msgs))
	}
}

func TestConsoleCollectorStop(t *testing.T) {
	mock, client := setup(t)

	mock.HandleFunc("Console.enable", map[string]interface{}{})
	mock.HandleFunc("Console.disable", map[string]interface{}{})

	collector := tools.NewConsoleCollector()
	ctx := context.Background()
	if err := collector.Start(ctx, client); err != nil {
		t.Fatalf("ConsoleCollector.Start: %v", err)
	}
	if err := collector.Stop(ctx, client); err != nil {
		t.Fatalf("ConsoleCollector.Stop: %v", err)
	}
}

func TestClearConsoleMessages(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("Console.clearMessages", map[string]interface{}{})

	ctx := context.Background()
	if err := tools.ClearConsoleMessages(ctx, client); err != nil {
		t.Fatalf("ClearConsoleMessages returned error: %v", err)
	}
}

func TestSetLogLevel(t *testing.T) {
	mock, client := setup(t)

	var receivedSource, receivedLevel string
	mock.Handle("Console.setLoggingChannelLevel", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			Source string `json:"source"`
			Level  string `json:"level"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedSource = p.Source
		receivedLevel = p.Level
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.SetLogLevel(ctx, client, "console", "warning"); err != nil {
		t.Fatalf("SetLogLevel returned error: %v", err)
	}
	if receivedSource != "console" {
		t.Errorf("expected source %q, got %q", "console", receivedSource)
	}
	if receivedLevel != "warning" {
		t.Errorf("expected level %q, got %q", "warning", receivedLevel)
	}
}

// ---------------------------------------------------------------------------
// Network standalone functions (network.go)
// ---------------------------------------------------------------------------

func TestGetResponseBody(t *testing.T) {
	mock, client := setup(t)
	_ = mock

	mock.HandleFunc("Network.getResponseBody", map[string]interface{}{
		"body":          `{"status":"ok"}`,
		"base64Encoded": false,
	})

	ctx := context.Background()
	body, encoded, err := tools.GetResponseBody(ctx, client, "req-1")
	if err != nil {
		t.Fatalf("GetResponseBody returned error: %v", err)
	}
	if body != `{"status":"ok"}` {
		t.Errorf("expected body %q, got %q", `{"status":"ok"}`, body)
	}
	if encoded {
		t.Errorf("expected base64Encoded=false")
	}
}

func TestSetExtraHeaders(t *testing.T) {
	mock, client := setup(t)

	var receivedHeaders map[string]interface{}
	mock.Handle("Network.setExtraHTTPHeaders", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			Headers map[string]interface{} `json:"headers"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedHeaders = p.Headers
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	headers := map[string]string{
		"X-Custom":      "value",
		"Authorization": "Bearer token",
	}
	if err := tools.SetExtraHeaders(ctx, client, headers); err != nil {
		t.Fatalf("SetExtraHeaders returned error: %v", err)
	}
	if receivedHeaders["X-Custom"] != "value" {
		t.Errorf("expected X-Custom=%q, got %v", "value", receivedHeaders["X-Custom"])
	}
	if receivedHeaders["Authorization"] != "Bearer token" {
		t.Errorf("expected Authorization=%q, got %v", "Bearer token", receivedHeaders["Authorization"])
	}
}

func TestSetRequestInterception(t *testing.T) {
	mock, client := setup(t)

	var receivedEnabled bool
	mock.Handle("Network.setInterceptionEnabled", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedEnabled = p.Enabled
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.SetRequestInterception(ctx, client, true); err != nil {
		t.Fatalf("SetRequestInterception returned error: %v", err)
	}
	if !receivedEnabled {
		t.Errorf("expected enabled=true")
	}
}

func TestInterceptContinue(t *testing.T) {
	mock, client := setup(t)

	var receivedRequestID string
	mock.Handle("Network.interceptContinue", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			RequestID string `json:"requestId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedRequestID = p.RequestID
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.InterceptContinue(ctx, client, "req-42"); err != nil {
		t.Fatalf("InterceptContinue returned error: %v", err)
	}
	if receivedRequestID != "req-42" {
		t.Errorf("expected requestId %q, got %q", "req-42", receivedRequestID)
	}
}

func TestInterceptWithResponse(t *testing.T) {
	mock, client := setup(t)

	var receivedRequestID string
	var receivedStatus float64
	var receivedBody string
	mock.Handle("Network.interceptWithResponse", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			RequestID  string  `json:"requestId"`
			StatusCode float64 `json:"statusCode"`
			Body       string  `json:"body"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedRequestID = p.RequestID
		receivedStatus = p.StatusCode
		receivedBody = p.Body
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	headers := map[string]string{"Content-Type": "text/plain"}
	if err := tools.InterceptWithResponse(ctx, client, "req-1", 200, headers, "OK"); err != nil {
		t.Fatalf("InterceptWithResponse returned error: %v", err)
	}
	if receivedRequestID != "req-1" {
		t.Errorf("expected requestId %q, got %q", "req-1", receivedRequestID)
	}
	if receivedStatus != 200 {
		t.Errorf("expected statusCode 200, got %v", receivedStatus)
	}
	if receivedBody != "OK" {
		t.Errorf("expected body %q, got %q", "OK", receivedBody)
	}
}

func TestSetEmulatedConditions(t *testing.T) {
	mock, client := setup(t)

	var receivedBandwidth float64
	var receivedLatency float64
	mock.Handle("Network.setEmulatedConditions", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			BytesPerSecondLimit float64 `json:"bytesPerSecondLimit"`
			LatencyMs           float64 `json:"latencyMs"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedBandwidth = p.BytesPerSecondLimit
		receivedLatency = p.LatencyMs
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.SetEmulatedConditions(ctx, client, 1000000, 100.5); err != nil {
		t.Fatalf("SetEmulatedConditions returned error: %v", err)
	}
	if receivedBandwidth != 1000000 {
		t.Errorf("expected bytesPerSecondLimit 1000000, got %v", receivedBandwidth)
	}
	if receivedLatency != 100.5 {
		t.Errorf("expected latencyMs 100.5, got %v", receivedLatency)
	}
}

func TestSetResourceCachingDisabled(t *testing.T) {
	mock, client := setup(t)

	var receivedDisabled bool
	mock.Handle("Network.setResourceCachingDisabled", func(_ string, params json.RawMessage) (interface{}, error) {
		var p struct {
			Disabled bool `json:"disabled"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		receivedDisabled = p.Disabled
		return map[string]interface{}{}, nil
	})

	ctx := context.Background()
	if err := tools.SetResourceCachingDisabled(ctx, client, true); err != nil {
		t.Fatalf("SetResourceCachingDisabled returned error: %v", err)
	}
	if !receivedDisabled {
		t.Errorf("expected disabled=true")
	}
}

func TestNetworkMonitorStop(t *testing.T) {
	mock, client := setup(t)

	mock.HandleFunc("Network.enable", map[string]interface{}{})
	mock.HandleFunc("Network.disable", map[string]interface{}{})

	monitor := tools.NewNetworkMonitor()
	ctx := context.Background()
	if err := monitor.Start(ctx, client); err != nil {
		t.Fatalf("NetworkMonitor.Start: %v", err)
	}
	if err := monitor.Stop(ctx, client); err != nil {
		t.Fatalf("NetworkMonitor.Stop: %v", err)
	}

	// Verify the monitor has no requests after a fresh start/stop cycle.
	reqs := monitor.GetRequests()
	if len(reqs) != 0 {
		t.Errorf("expected 0 requests after stop, got %d", len(reqs))
	}
	_ = mock
}
