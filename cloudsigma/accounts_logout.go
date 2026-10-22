package cloudsigma

import (
	"context"
	"errors"
	"net/http"
)

// ErrLogoutRedirect indicates that a redirect was refused and the initial POST
// outcome is unknown. Callers must not blindly retry.
var ErrLogoutRedirect = errors.New("logout redirect refused; outcome unknown")

// Logout sends the documented empty-object logout request for cookie auth.
// Callers manage authentication and the configured HTTP client's cookie jar;
// this method neither obtains cookies nor clears the jar or credentials.
// HTTP 200 is an empty acknowledgment, not a guarantee that all sessions or
// tokens have been invalidated. No retry, polling or follow-up is performed.
// Docs: https://docs.cloudsigma.com/en/latest/accounts.html
func (c *Client) Logout(ctx context.Context) (*Response, error) {
	req, err := c.NewRequest(http.MethodPost, "accounts/action/?do=logout", struct{}{})
	if err != nil {
		return nil, err
	}
	// Refuse redirects only for this action, without mutating caller policy.
	httpClient := c.guardedHTTPClient()
	httpClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return ErrLogoutRedirect
	}
	resp, err := c.do(ctx, req, nil, httpClient)
	if errors.Is(err, ErrLogoutRedirect) {
		// Do not expose net/http's URL-bearing redirect error.
		return nil, ErrLogoutRedirect
	}
	return resp, err
}
