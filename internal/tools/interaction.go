package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nnemirovsky/iwdp-mcp/internal/webkit"
)

// Click clicks the first element matching the given CSS selector.
func Click(ctx context.Context, client *webkit.Client, selector string) error {
	safeSelector, err := json.Marshal(selector)
	if err != nil {
		return fmt.Errorf("encoding selector: %w", err)
	}

	js := fmt.Sprintf(`(function() {
  var el = document.querySelector(%s);
  if (!el) throw new Error("no element matches selector " + %s);
  el.click();
})()`, safeSelector, safeSelector)

	_, err = EvaluateScript(ctx, client, js, false)
	return err
}

// Fill sets the value of an input element matching the given CSS selector
// and dispatches input and change events.
func Fill(ctx context.Context, client *webkit.Client, selector, value string) error {
	safeSelector, err := json.Marshal(selector)
	if err != nil {
		return fmt.Errorf("encoding selector: %w", err)
	}
	safeValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encoding value: %w", err)
	}

	js := fmt.Sprintf(`(function() {
  var el = document.querySelector(%s);
  if (!el) throw new Error("no element matches selector " + %s);
  var nativeInputValueSetter = Object.getOwnPropertyDescriptor(
    window.HTMLInputElement.prototype, 'value'
  );
  if (nativeInputValueSetter && nativeInputValueSetter.set) {
    nativeInputValueSetter.set.call(el, %s);
  } else {
    el.value = %s;
  }
  el.dispatchEvent(new Event('input', { bubbles: true }));
  el.dispatchEvent(new Event('change', { bubbles: true }));
})()`, safeSelector, safeSelector, safeValue, safeValue)

	_, err = EvaluateScript(ctx, client, js, false)
	return err
}

// TypeText dispatches keyboard events for each character on the currently
// focused element (document.activeElement).
func TypeText(ctx context.Context, client *webkit.Client, text string) error {
	safeText, err := json.Marshal(text)
	if err != nil {
		return fmt.Errorf("encoding text: %w", err)
	}

	js := fmt.Sprintf(`(function() {
  var el = document.activeElement;
  if (!el) throw new Error("no active element to type into");
  var text = %s;
  for (var i = 0; i < text.length; i++) {
    var ch = text[i];
    el.dispatchEvent(new KeyboardEvent('keydown', { key: ch, bubbles: true }));
    el.dispatchEvent(new KeyboardEvent('keypress', { key: ch, bubbles: true }));
    if (typeof el.value !== 'undefined') {
      el.value += ch;
      el.dispatchEvent(new Event('input', { bubbles: true }));
    }
    el.dispatchEvent(new KeyboardEvent('keyup', { key: ch, bubbles: true }));
  }
})()`, safeText)

	_, err = EvaluateScript(ctx, client, js, false)
	return err
}
