//go:build ignore

// ws-debug connects to an iwdp WebSocket and diagnoses Target routing.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ws-debug <ws-url>")
		os.Exit(1)
	}
	url := os.Args[1]
	fmt.Printf("Connecting to %s\n", url)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		fmt.Printf("DIAL ERROR: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Println("Connected!")

	// Read initial messages for 3 seconds (look for Target.targetCreated)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	fmt.Println("\n=== Reading initial messages (3s) ===")
	var targetID string
	for i := 0; i < 20; i++ {
		_, data, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("  Read stopped after %d messages: %v\n", i, err)
			break
		}
		fmt.Printf("  Message %d: %s\n", i, string(data))
		var msg struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(data, &msg) == nil && msg.Method == "Target.targetCreated" {
			var p struct {
				TargetInfo struct {
					TargetID string `json:"targetId"`
					Type     string `json:"type"`
				} `json:"targetInfo"`
			}
			json.Unmarshal(msg.Params, &p)
			fmt.Printf("  >>> TARGET: id=%s type=%s\n", p.TargetInfo.TargetID, p.TargetInfo.Type)
			if p.TargetInfo.Type == "page" {
				targetID = p.TargetInfo.TargetID
			}
		}
	}

	// Reset deadline
	conn.SetReadDeadline(time.Time{})

	// Try direct Runtime.evaluate (no Target wrapping)
	fmt.Println("\n=== Direct Runtime.evaluate (no Target wrapping) ===")
	direct := map[string]interface{}{
		"id":     1,
		"method": "Runtime.evaluate",
		"params": map[string]interface{}{"expression": "1+1", "returnByValue": true},
	}
	conn.WriteJSON(direct)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		fmt.Printf("  No response: %v\n", err)
	} else {
		fmt.Printf("  Response: %s\n", string(data))
	}
	conn.SetReadDeadline(time.Time{})

	// Try with Target wrapping if we have a targetID
	if targetID != "" {
		fmt.Printf("\n=== Target-wrapped Runtime.evaluate (targetId=%s) ===\n", targetID)
		inner, _ := json.Marshal(map[string]interface{}{
			"id":     2,
			"method": "Runtime.evaluate",
			"params": map[string]interface{}{"expression": "1+1", "returnByValue": true},
		})
		wrapped := map[string]interface{}{
			"id":     3,
			"method": "Target.sendMessageToTarget",
			"params": map[string]string{
				"targetId": targetID,
				"message":  string(inner),
			},
		}
		conn.WriteJSON(wrapped)
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		for i := 0; i < 5; i++ {
			_, data, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("  Read stopped: %v\n", err)
				break
			}
			fmt.Printf("  Response %d: %s\n", i, string(data))
		}
	} else {
		fmt.Println("\n=== No targetId received — cannot test Target wrapping ===")
	}

	fmt.Println("\nDone.")
}
