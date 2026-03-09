package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// Handler is a function that handles an incoming WebKit protocol message
// and returns the result or an error.
type Handler func(method string, params json.RawMessage) (interface{}, error)

// mockConn wraps a WebSocket connection with a write mutex to prevent
// concurrent writes from multiple goroutines (e.g., event sending + response sending).
type mockConn struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

func (mc *mockConn) writeMessage(messageType int, data []byte) error {
	mc.writeMu.Lock()
	defer mc.writeMu.Unlock()
	return mc.conn.WriteMessage(messageType, data)
}

// MockServer simulates a WebKit Inspector Protocol endpoint.
type MockServer struct {
	Server   *httptest.Server
	URL      string // WebSocket URL (ws://...)
	mu       sync.Mutex
	handlers map[string]Handler
	conns    []*mockConn
}

// NewMockServer creates and starts a mock WebSocket server.
func NewMockServer() *MockServer {
	m := &MockServer{
		handlers: make(map[string]Handler),
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		mc := &mockConn{conn: conn}
		m.mu.Lock()
		m.conns = append(m.conns, mc)
		m.mu.Unlock()

		go m.handleConn(mc)
	}))

	m.URL = "ws" + strings.TrimPrefix(m.Server.URL, "http")
	return m
}

// Handle registers a handler for a specific method.
func (m *MockServer) Handle(method string, handler Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[method] = handler
}

// HandleFunc registers a simple handler that returns a fixed result.
func (m *MockServer) HandleFunc(method string, result interface{}) {
	m.Handle(method, func(_ string, _ json.RawMessage) (interface{}, error) {
		return result, nil
	})
}

// SendEvent sends an event to all connected clients.
func (m *MockServer) SendEvent(method string, params interface{}) error {
	var rawParams json.RawMessage
	if params != nil {
		var err error
		rawParams, err = json.Marshal(params)
		if err != nil {
			return err
		}
	}

	msg := struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params,omitempty"`
	}{
		Method: method,
		Params: rawParams,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for _, mc := range m.conns {
		_ = mc.writeMessage(websocket.TextMessage, data)
	}
	return nil
}

// Close shuts down the mock server.
func (m *MockServer) Close() {
	m.mu.Lock()
	for _, mc := range m.conns {
		_ = mc.conn.Close()
	}
	m.conns = nil
	m.mu.Unlock()
	m.Server.Close()
}

func (m *MockServer) handleConn(mc *mockConn) {
	defer func() { _ = mc.conn.Close() }()
	for {
		_, data, err := mc.conn.ReadMessage()
		if err != nil {
			return
		}

		var msg struct {
			ID     int64           `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		m.mu.Lock()
		handler, ok := m.handlers[msg.Method]
		m.mu.Unlock()

		var resp []byte
		if ok {
			result, herr := handler(msg.Method, msg.Params)
			if herr != nil {
				resp, _ = json.Marshal(map[string]interface{}{
					"id": msg.ID,
					"error": map[string]interface{}{
						"code":    -32000,
						"message": herr.Error(),
					},
				})
			} else {
				resultJSON, _ := json.Marshal(result)
				resp, _ = json.Marshal(map[string]interface{}{
					"id":     msg.ID,
					"result": json.RawMessage(resultJSON),
				})
			}
		} else {
			// Default: return empty result
			resp, _ = json.Marshal(map[string]interface{}{
				"id":     msg.ID,
				"result": json.RawMessage("{}"),
			})
		}

		_ = mc.writeMessage(websocket.TextMessage, resp)
	}
}
