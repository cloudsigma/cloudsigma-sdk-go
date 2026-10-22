package cloudsigma

import (
	"context"
	"errors"
	"net/http"
)

// ErrSnapshotCloneRedirect indicates a refused snapshot clone redirect. The
// initial POST outcome is unknown; callers must not blindly retry.
var ErrSnapshotCloneRedirect = errors.New("snapshot clone redirect refused; outcome unknown")

// ErrBServerCreateRedirect indicates a refused backfilling creation redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrBServerCreateRedirect = errors.New("backfilling create redirect refused; outcome unknown")

// ErrBServerStartRedirect indicates a refused backfilling start redirect. The
// initial POST outcome is unknown; callers must not blindly retry.
var ErrBServerStartRedirect = errors.New("backfilling start redirect refused; outcome unknown")

// ErrBServerStopRedirect indicates a refused backfilling stop redirect. The
// initial POST outcome is unknown; callers must not blindly retry.
var ErrBServerStopRedirect = errors.New("backfilling stop redirect refused; outcome unknown")

// doResourceAction refuses redirects only for the four explicitly opted-in
// resource actions. Keep the effective transport, jar and timeout, without
// changing caller configuration or invoking its redirect callback.
func (c *Client) doResourceAction(ctx context.Context, req *http.Request, value interface{}, refusal error) (*Response, error) {
	httpClient := c.guardedHTTPClient()
	httpClient.Transport = resourceActionTransport{next: httpClient.Transport, refusal: refusal}
	httpClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return refusal }
	resp, err := c.do(ctx, req, value, httpClient)
	if errors.Is(err, refusal) {
		// The transport closes the redirect body before net/http parses Location.
		// Do not retain the net/http wrapper containing the initial request URL or its
		// response or an unacknowledged resource identity.
		return nil, refusal
	}
	return resp, err
}

// resourceActionTransport refuses redirect statuses before Client.Do can parse
// Location or process Set-Cookie. Even empty or malformed Location is refused.
// The caller's transport (including its TLS configuration) remains untouched.
type resourceActionTransport struct {
	next    http.RoundTripper
	refusal error
}

func (t resourceActionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.next.RoundTrip(req)
	if err != nil {
		// RoundTripper must return a nil response on error. If a custom transport
		// violates that contract, discard and close its otherwise orphaned body.
		// Preserve its error and cancellation/timeout semantics, not its response.
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, err
	}
	if resp != nil {
		switch resp.StatusCode {
		case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
			http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
			return nil, t.refusal
		}
	}
	return resp, nil
}
