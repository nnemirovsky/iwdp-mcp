package webkit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
	"github.com/nnemirovsky/iwdp-mcp/internal/webkit/testutil"
)

func TestClient_Send(t *testing.T) {
	mock := testutil.NewMockServer()
	defer mock.Close()

	mock.HandleFunc("Test.method", map[string]string{"key": "value"})

	ctx := context.Background()
	client, err := webkit.NewClient(ctx, mock.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	result, err := client.Send(ctx, "Test.method", nil)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got["key"] != "value" {
		t.Errorf("got %v, want key=value", got)
	}
}

func TestClient_SendWithParams(t *testing.T) {
	mock := testutil.NewMockServer()
	defer mock.Close()

	mock.Handle("Echo.params", func(_ string, params json.RawMessage) (interface{}, error) {
		return json.RawMessage(params), nil
	})

	ctx := context.Background()
	client, err := webkit.NewClient(ctx, mock.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	input := map[string]string{"hello": "world"}
	result, err := client.Send(ctx, "Echo.params", input)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got["hello"] != "world" {
		t.Errorf("got %v, want hello=world", got)
	}
}

func TestClient_SendError(t *testing.T) {
	mock := testutil.NewMockServer()
	defer mock.Close()

	mock.Handle("Fail.method", func(_ string, _ json.RawMessage) (interface{}, error) {
		return nil, fmt.Errorf("something went wrong")
	})

	ctx := context.Background()
	client, err := webkit.NewClient(ctx, mock.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.Send(ctx, "Fail.method", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "something went wrong" {
		t.Errorf("got error %q, want %q", err.Error(), "something went wrong")
	}
}

func TestClient_SendTimeout(t *testing.T) {
	mock := testutil.NewMockServer()
	defer mock.Close()

	// Register a handler that never responds — the mock server won't have this
	// method, so it returns empty result immediately. Instead we test context cancel.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	client, err := webkit.NewClient(context.Background(), mock.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Register a handler that blocks
	mock.Handle("Slow.method", func(_ string, _ json.RawMessage) (interface{}, error) {
		time.Sleep(5 * time.Second)
		return nil, nil
	})

	_, err = client.Send(ctx, "Slow.method", nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestClient_Events(t *testing.T) {
	mock := testutil.NewMockServer()
	defer mock.Close()

	ctx := context.Background()
	client, err := webkit.NewClient(ctx, mock.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	var mu sync.Mutex
	var received []string

	client.OnEvent("Console.messageAdded", func(method string, params json.RawMessage) {
		mu.Lock()
		received = append(received, method)
		mu.Unlock()
	})

	// Send an event from the mock server
	if err := mock.SendEvent("Console.messageAdded", map[string]string{"text": "hello"}); err != nil {
		t.Fatalf("SendEvent: %v", err)
	}

	// Wait for event to be received
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}
	if received[0] != "Console.messageAdded" {
		t.Errorf("got event %q, want Console.messageAdded", received[0])
	}
}

func TestClient_GlobalEventHandler(t *testing.T) {
	mock := testutil.NewMockServer()
	defer mock.Close()

	ctx := context.Background()
	client, err := webkit.NewClient(ctx, mock.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	var mu sync.Mutex
	var received []string

	client.OnAnyEvent(func(method string, params json.RawMessage) {
		mu.Lock()
		received = append(received, method)
		mu.Unlock()
	})

	if err := mock.SendEvent("Page.loadEventFired", nil); err != nil {
		t.Fatalf("SendEvent: %v", err)
	}
	if err := mock.SendEvent("DOM.documentUpdated", nil); err != nil {
		t.Fatalf("SendEvent: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Fatalf("expected 2 events, got %d", len(received))
	}
}

func TestClient_ConcurrentSends(t *testing.T) {
	mock := testutil.NewMockServer()
	defer mock.Close()

	mock.Handle("Test.echo", func(_ string, params json.RawMessage) (interface{}, error) {
		return json.RawMessage(params), nil
	})

	ctx := context.Background()
	client, err := webkit.NewClient(ctx, mock.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	var wg sync.WaitGroup
	errs := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			input := map[string]int{"n": n}
			result, err := client.Send(ctx, "Test.echo", input)
			if err != nil {
				errs <- fmt.Errorf("send %d: %w", n, err)
				return
			}
			var got map[string]int
			if err := json.Unmarshal(result, &got); err != nil {
				errs <- fmt.Errorf("unmarshal %d: %w", n, err)
				return
			}
			if got["n"] != n {
				errs <- fmt.Errorf("got n=%d, want %d", got["n"], n)
			}
		}(i)
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func TestClient_Close(t *testing.T) {
	mock := testutil.NewMockServer()
	defer mock.Close()

	ctx := context.Background()
	client, err := webkit.NewClient(ctx, mock.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Double close should be safe
	if err := client.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}

	// Send after close should fail
	_, err = client.Send(ctx, "Test.method", nil)
	if err == nil {
		t.Fatal("expected error after close")
	}
}

func TestClient_EnableDisableDomain(t *testing.T) {
	mock := testutil.NewMockServer()
	defer mock.Close()

	var enabled []string
	var mu sync.Mutex

	mock.Handle("Page.enable", func(method string, _ json.RawMessage) (interface{}, error) {
		mu.Lock()
		enabled = append(enabled, method)
		mu.Unlock()
		return nil, nil
	})
	mock.Handle("Page.disable", func(method string, _ json.RawMessage) (interface{}, error) {
		mu.Lock()
		enabled = append(enabled, method)
		mu.Unlock()
		return nil, nil
	})

	ctx := context.Background()
	client, err := webkit.NewClient(ctx, mock.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.Enable(ctx, "Page"); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if err := client.Disable(ctx, "Page"); err != nil {
		t.Fatalf("Disable: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(enabled) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(enabled))
	}
}
