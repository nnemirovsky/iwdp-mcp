package webkit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// EventHandler is a callback for protocol events.
type EventHandler func(method string, params json.RawMessage)

// Client communicates with a WebKit Inspector Protocol endpoint over WebSocket.
// When connected through ios-webkit-debug-proxy, messages are automatically
// wrapped/unwrapped using Target.sendMessageToTarget routing.
type Client struct {
	conn    *websocket.Conn
	url     string
	nextID  atomic.Int64
	writeMu sync.Mutex // protects conn.WriteMessage (gorilla/websocket is not concurrent-write-safe)
	mu      sync.Mutex
	pending map[int64]chan *Message

	eventMu        sync.RWMutex
	handlers       map[string][]EventHandler
	globalHandlers []EventHandler

	done      chan struct{}
	closeOnce sync.Once

	// Target-based routing (used by ios-webkit-debug-proxy)
	targetID    string
	targetReady chan struct{}

	// Dialer allows overriding the WebSocket dialer for testing.
	Dialer *websocket.Dialer
}

// TargetWaitTimeout controls how long NewClient waits for a Target.targetCreated
// event before falling back to direct (non-Target) mode. Default is 100ms which
// is enough for local iwdp connections. Increase for slower environments (CI).
var TargetWaitTimeout = 100 * time.Millisecond

// NewClient creates a new WebKit Inspector Protocol client connected to the given WebSocket URL.
// If the endpoint uses Target-based routing (ios-webkit-debug-proxy), the client
// automatically wraps/unwraps messages via Target.sendMessageToTarget.
func NewClient(ctx context.Context, wsURL string) (*Client, error) {
	return NewClientWithDialer(ctx, wsURL, websocket.DefaultDialer)
}

// NewClientWithDialer creates a client with a custom WebSocket dialer.
func NewClientWithDialer(ctx context.Context, wsURL string, dialer *websocket.Dialer) (*Client, error) {
	c := &Client{
		url:         wsURL,
		pending:     make(map[int64]chan *Message),
		handlers:    make(map[string][]EventHandler),
		done:        make(chan struct{}),
		targetReady: make(chan struct{}),
		Dialer:      dialer,
	}
	if err := c.connect(ctx); err != nil {
		return nil, err
	}
	go c.readLoop()

	// Wait briefly for Target.targetCreated — if received, enable Target routing.
	select {
	case <-c.targetReady:
		// Target routing enabled
	case <-time.After(TargetWaitTimeout):
		// No Target event — use direct mode (e.g., mock servers in tests)
	case <-ctx.Done():
		_ = c.Close()
		return nil, ctx.Err()
	}

	return c, nil
}

// maxReadSize is the maximum WebSocket message size the client will accept (512 MB).
// Heap snapshots on heavy pages can be 50-200+ MB, and Target-based routing adds
// ~30% overhead from JSON string escaping, so we need a generous limit.
const maxReadSize = 512 * 1024 * 1024

func (c *Client) connect(ctx context.Context) error {
	conn, _, err := c.Dialer.DialContext(ctx, c.url, nil)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", c.url, err)
	}
	conn.SetReadLimit(maxReadSize)
	c.conn = conn
	return nil
}

// Close closes the WebSocket connection.
// Pending Send calls detect closure via the done channel and return an error.
func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.done)
		if c.conn != nil {
			err = c.conn.Close()
		}
	})
	return err
}

// Send sends a method call and waits for the response.
// If Target routing is active, the message is automatically wrapped.
func (c *Client) Send(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	// Early check: if already closed, fail fast.
	select {
	case <-c.done:
		return nil, fmt.Errorf("connection closed")
	default:
	}

	innerID := c.nextID.Add(1)

	var rawParams json.RawMessage
	if params != nil {
		var err error
		rawParams, err = json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshaling params: %w", err)
		}
	}

	innerMsg := Message{
		ID:     innerID,
		Method: method,
		Params: rawParams,
	}

	// Build the wire message — either wrapped in Target or sent directly.
	// Read targetID under lock to avoid data race with handleTargetCreated.
	c.mu.Lock()
	tid := c.targetID
	c.mu.Unlock()

	var wireData []byte
	if tid != "" {
		innerData, err := json.Marshal(innerMsg)
		if err != nil {
			return nil, fmt.Errorf("marshaling inner message: %w", err)
		}
		outerID := c.nextID.Add(1)
		outer := Message{
			ID:     outerID,
			Method: "Target.sendMessageToTarget",
		}
		outerParams, _ := json.Marshal(map[string]string{
			"targetId": tid,
			"message":  string(innerData),
		})
		outer.Params = outerParams
		wireData, err = json.Marshal(outer)
		if err != nil {
			return nil, fmt.Errorf("marshaling outer message: %w", err)
		}
	} else {
		var err error
		wireData, err = json.Marshal(innerMsg)
		if err != nil {
			return nil, fmt.Errorf("marshaling message: %w", err)
		}
	}

	ch := make(chan *Message, 1)
	c.mu.Lock()
	c.pending[innerID] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, innerID)
		c.mu.Unlock()
	}()

	c.writeMu.Lock()
	err := c.conn.WriteMessage(websocket.TextMessage, wireData)
	c.writeMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("writing message: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, fmt.Errorf("connection closed")
	case resp := <-ch:
		if resp == nil {
			return nil, fmt.Errorf("connection closed while waiting for response")
		}
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	}
}

// SendWithTimeout sends a method call with a timeout.
func (c *Client) SendWithTimeout(method string, params interface{}, timeout time.Duration) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.Send(ctx, method, params)
}

// OnEvent registers a handler for a specific event method.
func (c *Client) OnEvent(method string, handler EventHandler) {
	c.eventMu.Lock()
	defer c.eventMu.Unlock()
	c.handlers[method] = append(c.handlers[method], handler)
}

// OnAnyEvent registers a handler that receives all events.
func (c *Client) OnAnyEvent(handler EventHandler) {
	c.eventMu.Lock()
	defer c.eventMu.Unlock()
	c.globalHandlers = append(c.globalHandlers, handler)
}

func (c *Client) readLoop() {
	for {
		select {
		case <-c.done:
			return
		default:
		}

		_, data, err := c.conn.ReadMessage()
		if err != nil {
			// Connection closed or error — stop reading
			c.closeOnce.Do(func() {
				close(c.done)
			})
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		// Handle Target-based routing from iwdp
		switch msg.Method {
		case "Target.targetCreated":
			c.handleTargetCreated(msg.Params)
			continue
		case "Target.dispatchMessageFromTarget":
			c.handleDispatchFromTarget(msg.Params)
			continue
		}

		if msg.ID > 0 {
			// Response to a request
			c.mu.Lock()
			ch, ok := c.pending[msg.ID]
			c.mu.Unlock()
			if ok {
				select {
				case ch <- &msg:
				case <-c.done:
				}
			}
		} else if msg.Method != "" {
			// Event
			c.dispatchEvent(msg.Method, msg.Params)
		}
	}
}

func (c *Client) handleTargetCreated(params json.RawMessage) {
	var p struct {
		TargetInfo struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
		} `json:"targetInfo"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}
	if p.TargetInfo.TargetID != "" && p.TargetInfo.Type == "page" {
		c.mu.Lock()
		c.targetID = p.TargetInfo.TargetID
		c.mu.Unlock()
		select {
		case <-c.targetReady:
			// already closed
		default:
			close(c.targetReady)
		}
	}
}

func (c *Client) handleDispatchFromTarget(params json.RawMessage) {
	var p struct {
		TargetID string `json:"targetId"`
		Message  string `json:"message"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}

	var innerMsg Message
	if err := json.Unmarshal([]byte(p.Message), &innerMsg); err != nil {
		return
	}

	if innerMsg.ID > 0 {
		// Response to a request — route by inner ID
		c.mu.Lock()
		ch, ok := c.pending[innerMsg.ID]
		c.mu.Unlock()
		if ok {
			select {
			case ch <- &innerMsg:
			case <-c.done:
			}
		}
	} else if innerMsg.Method != "" {
		// Event from the target (e.g., Console.messageAdded, Network.requestWillBeSent)
		c.dispatchEvent(innerMsg.Method, innerMsg.Params)
	}
}

func (c *Client) dispatchEvent(method string, params json.RawMessage) {
	c.eventMu.RLock()
	// Copy slices so handlers can safely call OnEvent/OnAnyEvent without deadlock or race.
	handlers := make([]EventHandler, len(c.handlers[method]))
	copy(handlers, c.handlers[method])
	globals := make([]EventHandler, len(c.globalHandlers))
	copy(globals, c.globalHandlers)
	c.eventMu.RUnlock()

	for _, h := range handlers {
		h(method, params)
	}
	for _, h := range globals {
		h(method, params)
	}
}

// Done returns a channel that is closed when the connection is closed.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// URL returns the WebSocket URL this client is connected to.
func (c *Client) URL() string {
	return c.url
}
