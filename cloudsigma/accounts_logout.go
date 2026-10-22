package cloudsigma

import (
	"context"
	"net/http"
)

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
	return c.Do(ctx, req, nil)
}
