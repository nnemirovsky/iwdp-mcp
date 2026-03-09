package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// AnimationEnable enables the Animation domain.
func AnimationEnable(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Animation.enable", nil)
	return err
}

// AnimationDisable disables the Animation domain.
func AnimationDisable(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Animation.disable", nil)
	return err
}

// --- Animation Tracking Collector ---

// AnimationTrackingUpdate represents a single Animation tracking update.
type AnimationTrackingUpdate struct {
	TrackingAnimationID string `json:"trackingAnimationId"`
	AnimationState      string `json:"animationState"` // ready, delayed, active, canceled, done
	NodeID              int    `json:"nodeId,omitempty"`
	AnimationName       string `json:"animationName,omitempty"`
	TransitionProperty  string `json:"transitionProperty,omitempty"`
}

// AnimationTrackingEvent represents a timestamped animation tracking event.
type AnimationTrackingEvent struct {
	Timestamp float64                 `json:"timestamp"`
	Event     AnimationTrackingUpdate `json:"event"`
}

// AnimationTrackingResult holds the collected animation tracking data.
type AnimationTrackingResult struct {
	Events []AnimationTrackingEvent `json:"events"`
}

// AnimationTrackingCollector collects Animation tracking events.
type AnimationTrackingCollector struct {
	mu      sync.Mutex
	events  []AnimationTrackingEvent
	started bool
	done    chan struct{}
}

// NewAnimationTrackingCollector creates a new AnimationTrackingCollector.
func NewAnimationTrackingCollector() *AnimationTrackingCollector {
	return &AnimationTrackingCollector{}
}

// Start begins animation tracking, collecting trackingUpdate events.
func (c *AnimationTrackingCollector) Start(ctx context.Context, client *webkit.Client) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}
	c.started = true
	c.events = nil
	c.done = make(chan struct{})
	c.mu.Unlock()

	client.OnEvent("Animation.trackingUpdate", func(method string, params json.RawMessage) {
		var evt AnimationTrackingEvent
		if err := json.Unmarshal(params, &evt); err != nil {
			return
		}
		c.mu.Lock()
		c.events = append(c.events, evt)
		if len(c.events) > maxCollectorEntries {
			c.events = c.events[len(c.events)-maxCollectorEntries:]
		}
		c.mu.Unlock()
	})

	client.OnEvent("Animation.trackingComplete", func(method string, params json.RawMessage) {
		c.mu.Lock()
		ch := c.done
		c.mu.Unlock()
		if ch != nil {
			select {
			case <-ch:
			default:
				close(ch)
			}
		}
	})

	_, err := client.Send(ctx, "Animation.startTracking", nil)
	return err
}

// Stop stops animation tracking and returns the collected events.
func (c *AnimationTrackingCollector) Stop(ctx context.Context, client *webkit.Client) (*AnimationTrackingResult, error) {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return &AnimationTrackingResult{}, nil
	}
	ch := c.done
	c.mu.Unlock()

	_, err := client.Send(ctx, "Animation.stopTracking", nil)
	if err != nil {
		return nil, err
	}

	// Wait for trackingComplete event.
	if ch != nil {
		select {
		case <-ch:
		case <-ctx.Done():
		}
	}

	c.mu.Lock()
	c.started = false
	result := &AnimationTrackingResult{
		Events: make([]AnimationTrackingEvent, len(c.events)),
	}
	copy(result.Events, c.events)
	c.events = nil
	c.mu.Unlock()

	return result, nil
}

// GetAnimationEffect requests the effect target for an animation and returns the raw result.
func GetAnimationEffect(ctx context.Context, client *webkit.Client, animationID string) (json.RawMessage, error) {
	result, err := client.Send(ctx, "Animation.requestEffectTarget", map[string]interface{}{
		"animationId": animationID,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ResolveAnimation resolves an animation to a remote object for further inspection.
func ResolveAnimation(ctx context.Context, client *webkit.Client, animationID string, objectGroup string) (*webkit.RemoteObject, error) {
	result, err := client.Send(ctx, "Animation.resolveAnimation", map[string]interface{}{
		"animationId": animationID,
		"objectGroup": objectGroup,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Object webkit.RemoteObject `json:"object"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("decoding animation object: %w", err)
	}
	return &resp.Object, nil
}
