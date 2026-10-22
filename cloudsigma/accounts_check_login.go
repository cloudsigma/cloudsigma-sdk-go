package cloudsigma

import (
	"context"
	"errors"
	"net/http"
)

// ErrCheckLoginRedirect indicates that a redirect was refused and the initial POST
// outcome is unknown. Callers must not blindly retry.
var ErrCheckLoginRedirect = errors.New("login check redirect refused; outcome unknown")

// CheckLogin sends the documented empty-object authentication check using this
// client's existing authentication configuration. HTTP 200 with an empty body is
// documented as no error. It does not log in, acquire or manage cookies, perform
// OTP authentication, or establish a durable session or authorization guarantee.
// No retries, polling or follow-up requests are performed.
// Docs: https://docs.cloudsigma.com/en/latest/accounts.html
func (c *Client) CheckLogin(ctx context.Context) (*Response, error) {
	req, err := c.NewRequest(http.MethodPost, "accounts/action/?do=check_login", struct{}{})
	if err != nil {
		return nil, err
	}
	return c.doResourceAction(ctx, req, nil, ErrCheckLoginRedirect)
}
