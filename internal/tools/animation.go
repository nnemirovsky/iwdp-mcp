package tools

import (
	"context"
	"encoding/json"
	"fmt"

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

// AnimationStartTracking starts tracking animations.
func AnimationStartTracking(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Animation.startTracking", nil)
	return err
}

// AnimationStopTracking stops tracking animations.
func AnimationStopTracking(ctx context.Context, client *webkit.Client) error {
	_, err := client.Send(ctx, "Animation.stopTracking", nil)
	return err
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
