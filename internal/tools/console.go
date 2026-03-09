package tools

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

const maxCollectorEntries = 10000

// ConsoleCollector collects console messages via events.
type ConsoleCollector struct {
	mu       sync.Mutex
	messages []webkit.ConsoleMessage
	started  bool
}

// NewConsoleCollector creates a new ConsoleCollector.
func NewConsoleCollector() *ConsoleCollector {
	return &ConsoleCollector{}
}

// Start enables the Console domain and registers a handler for Console.messageAdded events.
// It is idempotent: calling Start multiple times will not register duplicate handlers.
func (c *ConsoleCollector) Start(ctx context.Context, client *webkit.Client) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}
	c.started = true
	c.mu.Unlock()

	client.OnEvent("Console.messageAdded", func(method string, params json.RawMessage) {
		var envelope struct {
			Message webkit.ConsoleMessage `json:"message"`
		}
		if err := json.Unmarshal(params, &envelope); err != nil {
			return
		}
		c.mu.Lock()
		c.messages = append(c.messages, envelope.Message)
		if len(c.messages) > maxCollectorEntries {
			c.messages = c.messages[len(c.messages)-maxCollectorEntries:]
		}
		c.mu.Unlock()
	})

	_, err := client.Send(ctx, "Console.enable", nil)
	return err
}

// Stop disables the Console domain.
func (c *ConsoleCollector) Stop(ctx context.Context, client *webkit.Client) error {
	c.mu.Lock()
	c.started = false
	c.mu.Unlock()
	_, err := client.Send(ctx, "Console.disable", nil)
	return err
}

// GetMessages returns a copy of the collected console messages.
func (c *ConsoleCollector) GetMessages() []webkit.ConsoleMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	msgs := make([]webkit.ConsoleMessage, len(c.messages))
	copy(msgs, c.messages)
	return msgs
}

// Clear clears all collected console messages.
func (c *ConsoleCollector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = nil
}

// ClearConsoleMessages sends a command to clear the console in the inspected page.
func ClearConsoleMessages(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Console.clearMessages", nil)
	return err
}

// SetLogLevel sets the logging channel level for a given source.
func SetLogLevel(ctx context.Context, client *webkit.Client, source string, level string) error {
	_, err := client.Send(ctx, "Console.setLoggingChannelLevel", map[string]string{
		"source": source,
		"level":  level,
	})
	return err
}
