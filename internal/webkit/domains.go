package webkit

import (
	"context"
	"time"
)

const defaultTimeout = 30 * time.Second

// Enable enables a WebKit Inspector Protocol domain.
func (c *Client) Enable(ctx context.Context, domain string) error {
	_, err := c.Send(ctx, domain+".enable", nil)
	return err
}

// Disable disables a WebKit Inspector Protocol domain.
func (c *Client) Disable(ctx context.Context, domain string) error {
	_, err := c.Send(ctx, domain+".disable", nil)
	return err
}

// EnableDomain enables a domain using the default timeout.
func (c *Client) EnableDomain(domain string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return c.Enable(ctx, domain)
}

// DisableDomain disables a domain using the default timeout.
func (c *Client) DisableDomain(domain string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return c.Disable(ctx, domain)
}
