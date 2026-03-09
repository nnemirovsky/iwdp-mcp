package webkit

import (
	"encoding/json"
	"strconv"
)

// Message is a generic WebKit Inspector Protocol message (request or event).
type Message struct {
	ID     int64           `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *ErrorData      `json:"error,omitempty"`
}

// ErrorData represents a protocol error response.
type ErrorData struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *ErrorData) Error() string {
	if len(e.Data) > 0 {
		return e.Message + ": " + string(e.Data)
	}
	return e.Message
}

// DeviceEntry represents a device from iwdp's /json endpoint on the listing port.
type DeviceEntry struct {
	DeviceID   string `json:"deviceId"`
	DeviceName string `json:"deviceName"`
	URL        string `json:"url"`
}

// PageEntry represents a page/tab from iwdp's /json endpoint on a device port.
type PageEntry struct {
	DevToolsFrontendURL  string  `json:"devtoolsFrontendUrl"`
	FaviconURL           string  `json:"faviconUrl"`
	ThumbnailURL         string  `json:"thumbnailUrl"`
	Title                string  `json:"title"`
	Type                 string  `json:"type"`
	URL                  string  `json:"url"`
	WebSocketDebuggerURL string  `json:"webSocketDebuggerUrl"`
	PageID               FlexInt `json:"appId"`
}

// FlexInt unmarshals from both JSON integers and strings.
// iwdp returns appId as int in some versions and string in others.
type FlexInt int

func (fi *FlexInt) UnmarshalJSON(data []byte) error {
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		*fi = FlexInt(i)
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		// Non-numeric string (e.g. "com.apple.mobilesafari") — use 0.
		*fi = 0
		return nil
	}
	*fi = FlexInt(n)
	return nil
}

// --- DOM types ---

// Node represents a DOM node.
type Node struct {
	NodeID          int      `json:"nodeId"`
	NodeType        int      `json:"nodeType"`
	NodeName        string   `json:"nodeName"`
	LocalName       string   `json:"localName"`
	NodeValue       string   `json:"nodeValue"`
	ChildCount      int      `json:"childNodeCount,omitempty"`
	Children        []*Node  `json:"children,omitempty"`
	Attributes      []string `json:"attributes,omitempty"`
	DocumentURL     string   `json:"documentURL,omitempty"`
	FrameID         string   `json:"frameId,omitempty"`
	ContentDocument *Node    `json:"contentDocument,omitempty"`
}

// --- Runtime types ---

// RemoteObject represents a mirror object referencing original JavaScript objects.
type RemoteObject struct {
	Type        string          `json:"type"`
	Subtype     string          `json:"subtype,omitempty"`
	ClassName   string          `json:"className,omitempty"`
	Value       json.RawMessage `json:"value,omitempty"`
	Description string          `json:"description,omitempty"`
	ObjectID    string          `json:"objectId,omitempty"`
	Preview     *ObjectPreview  `json:"preview,omitempty"`
}

// ObjectPreview provides a preview of an object.
type ObjectPreview struct {
	Type        string            `json:"type"`
	Subtype     string            `json:"subtype,omitempty"`
	Description string            `json:"description,omitempty"`
	Overflow    bool              `json:"overflow"`
	Properties  []PropertyPreview `json:"properties,omitempty"`
}

// PropertyPreview is a preview of a single property.
type PropertyPreview struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value,omitempty"`
}

// ExceptionDetails holds information about an exception during evaluation.
type ExceptionDetails struct {
	Text      string        `json:"text"`
	Line      int           `json:"line"`
	Column    int           `json:"column"`
	URL       string        `json:"url,omitempty"`
	Exception *RemoteObject `json:"exception,omitempty"`
}

// PropertyDescriptor describes a property of a remote object.
type PropertyDescriptor struct {
	Name         string        `json:"name"`
	Value        *RemoteObject `json:"value,omitempty"`
	Writable     bool          `json:"writable,omitempty"`
	Get          *RemoteObject `json:"get,omitempty"`
	Set          *RemoteObject `json:"set,omitempty"`
	Configurable bool          `json:"configurable"`
	Enumerable   bool          `json:"enumerable"`
	IsOwn        bool          `json:"isOwn,omitempty"`
}

// --- CSS types ---

// CSSRule represents a CSS rule.
type CSSRule struct {
	RuleID       json.RawMessage `json:"ruleId,omitempty"`
	SelectorList *SelectorList   `json:"selectorList,omitempty"`
	Origin       string          `json:"origin"`
	Style        *CSSStyle       `json:"style"`
}

// SelectorList is a list of selectors.
type SelectorList struct {
	Selectors []Selector `json:"selectors"`
	Text      string     `json:"text"`
}

// Selector is a CSS selector.
type Selector struct {
	Text string `json:"text"`
}

// CSSStyle represents a CSS style declaration.
type CSSStyle struct {
	StyleID    json.RawMessage `json:"styleId,omitempty"`
	Properties []CSSProperty   `json:"cssProperties"`
	Text       string          `json:"cssText,omitempty"`
}

// CSSProperty is a single CSS property.
type CSSProperty struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Priority string `json:"priority,omitempty"`
	Implicit bool   `json:"implicit,omitempty"`
	Status   string `json:"status,omitempty"`
}

// CSSStyleSheet represents a stylesheet.
type CSSStyleSheet struct {
	StyleSheetID string `json:"styleSheetId"`
	FrameID      string `json:"frameId"`
	SourceURL    string `json:"sourceURL"`
	Origin       string `json:"origin"`
	Title        string `json:"title"`
	Disabled     bool   `json:"disabled"`
	IsInline     bool   `json:"isInline"`
}

// --- Network types ---

// NetworkRequest represents a captured network request.
type NetworkRequest struct {
	RequestID string            `json:"requestId"`
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	Timestamp float64           `json:"timestamp"`
	Type      string            `json:"type,omitempty"`
}

// NetworkResponse represents a network response.
type NetworkResponse struct {
	URL        string            `json:"url"`
	Status     int               `json:"status"`
	StatusText string            `json:"statusText"`
	Headers    map[string]string `json:"headers"`
	MimeType   string            `json:"mimeType"`
}

// --- Console types ---

// ConsoleMessage represents a console log message.
type ConsoleMessage struct {
	Source     string         `json:"source"`
	Level      string         `json:"level"`
	Text       string         `json:"text"`
	URL        string         `json:"url,omitempty"`
	Line       int            `json:"line,omitempty"`
	Column     int            `json:"column,omitempty"`
	Parameters []RemoteObject `json:"parameters,omitempty"`
}

// --- Cookie types ---

// Cookie represents an HTTP cookie.
type Cookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	Session  bool    `json:"session"`
	SameSite string  `json:"sameSite,omitempty"`
}

// --- DOMStorage types ---

// StorageID identifies a storage area.
type StorageID struct {
	SecurityOrigin string `json:"securityOrigin"`
	IsLocalStorage bool   `json:"isLocalStorage"`
}

// StorageItem is a key-value pair from DOM storage.
type StorageItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// --- IndexedDB types ---

// DatabaseWithObjectStores describes an IndexedDB database.
type DatabaseWithObjectStores struct {
	Name         string        `json:"name"`
	Version      int           `json:"version"`
	ObjectStores []ObjectStore `json:"objectStores"`
}

// ObjectStore describes an IndexedDB object store.
type ObjectStore struct {
	Name          string      `json:"name"`
	KeyPath       interface{} `json:"keyPath"`
	AutoIncrement bool        `json:"autoIncrement"`
	Indexes       []Index     `json:"indexes"`
}

// Index describes an IndexedDB index.
type Index struct {
	Name       string      `json:"name"`
	KeyPath    interface{} `json:"keyPath"`
	Unique     bool        `json:"unique"`
	MultiEntry bool        `json:"multiEntry"`
}

// --- Debugger types ---

// Location identifies a position in a script.
type Location struct {
	ScriptID     string `json:"scriptId"`
	LineNumber   int    `json:"lineNumber"`
	ColumnNumber int    `json:"columnNumber,omitempty"`
}

// BreakpointID is the identifier for a breakpoint.
type BreakpointID string

// CallFrame represents a call stack frame.
type CallFrame struct {
	CallFrameID  string       `json:"callFrameId"`
	FunctionName string       `json:"functionName"`
	Location     Location     `json:"location"`
	ScopeChain   []Scope      `json:"scopeChain"`
	This         RemoteObject `json:"this"`
}

// Scope describes a scope in a call frame.
type Scope struct {
	Type   string       `json:"type"`
	Object RemoteObject `json:"object"`
}

// --- Timeline types ---

// TimelineEvent represents a timeline record.
type TimelineEvent struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data,omitempty"`
	Children  []TimelineEvent `json:"children,omitempty"`
	StartTime float64         `json:"startTime,omitempty"`
	EndTime   float64         `json:"endTime,omitempty"`
}

// --- Layer types ---

// Layer represents a compositing layer.
type Layer struct {
	LayerID          string `json:"layerId"`
	NodeID           int    `json:"nodeId,omitempty"`
	Bounds           Rect   `json:"bounds"`
	PaintCount       int    `json:"paintCount"`
	Memory           int    `json:"memory,omitempty"`
	CompositedBounds Rect   `json:"compositedBounds,omitempty"`
}

// Rect is a simple rectangle.
type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// --- Heap types ---

// HeapSnapshotData holds a heap snapshot result.
type HeapSnapshotData struct {
	SnapshotData string `json:"snapshotData"`
}

// --- Animation types ---

// AnimationPayload describes a web animation.
type AnimationPayload struct {
	AnimationID string          `json:"animationId"`
	Name        string          `json:"name,omitempty"`
	CSSName     string          `json:"cssAnimationName,omitempty"`
	Effect      json.RawMessage `json:"effect,omitempty"`
}

// --- Canvas types ---

// CanvasEntry describes a canvas context.
type CanvasEntry struct {
	CanvasID    string `json:"canvasId"`
	FrameID     string `json:"frameId"`
	ContextType string `json:"contextType"`
}
