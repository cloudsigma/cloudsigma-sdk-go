package cloudsigma_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type tunnelDeleteWire struct {
	t                   *testing.T
	body                string
	code, calls, closed int
	err                 error
	basic               bool
}
type tunnelDeleteBody struct {
	*strings.Reader
	w *tunnelDeleteWire
}

func (b *tunnelDeleteBody) Close() error { b.w.closed++; return nil }
func (w *tunnelDeleteWire) RoundTrip(q *http.Request) (*http.Response, error) {
	w.calls++
	if q.Method != "DELETE" || q.URL.RequestURI() != "/custom/api/2.0/tunnels/synthetic-tunnel/" {
		w.t.Fatalf("route %s %s", q.Method, q.URL)
	}
	if w.basic {
		u, p, ok := q.BasicAuth()
		if !ok || u != "synthetic-user" || p != "synthetic-password" {
			w.t.Fatal("basic header")
		}
	} else if q.Header.Get("Authorization") != "Bearer synthetic" {
		w.t.Fatal("authentication")
	}
	if q.Body != nil {
		b, e := io.ReadAll(q.Body)
		if e != nil || len(b) != 0 {
			w.t.Fatal("request body")
		}
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if w.err != nil {
		return nil, w.err
	}
	return &http.Response{StatusCode: w.code, Header: make(http.Header), Body: &tunnelDeleteBody{strings.NewReader(w.body), w}, Request: q}, nil
}
func tunnelDeleteClient(w *tunnelDeleteWire) *sdk.Client {
	return sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
}
func TestTunnelsDeleteAcknowledgment(t *testing.T) {
	w := &tunnelDeleteWire{t: t, code: 204}
	r, e := tunnelDeleteClient(w).Tunnels.Delete(context.Background(), "synthetic-tunnel")
	if e != nil || r == nil || r.StatusCode != 204 || r.Meta != nil || w.calls != 1 || w.closed != 1 {
		t.Fatalf("acknowledgment %v %v %+v", r, e, w)
	}
	// One request, no retry or polling; no VPN deletion completion is inferred.
}
func TestTunnelsDeleteRefusals(t *testing.T) {
	for _, id := range []string{"", ".", "..", "a/b", "a?x", "a#x", "%2f", "a\n"} {
		w := &tunnelDeleteWire{t: t}
		r, e := tunnelDeleteClient(w).Tunnels.Delete(context.Background(), id)
		if e == nil || r != nil || w.calls != 0 {
			t.Fatalf("invalid identifier %q", id)
		}
	}
}
func TestTunnelsDeleteErrors(t *testing.T) {
	for _, code := range []int{400, 401, 403, 404, 409, 429, 500} {
		w := &tunnelDeleteWire{t: t, code: code, body: `[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`}
		r, e := tunnelDeleteClient(w).Tunnels.Delete(context.Background(), "synthetic-tunnel")
		var api *sdk.ErrorResponse
		if r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0].Message != "synthetic refusal" || w.calls != 1 || w.closed != 1 {
			t.Fatalf("HTTP%d: %v", code, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	w := &tunnelDeleteWire{t: t, err: sentinel}
	r, e := tunnelDeleteClient(w).Tunnels.Delete(context.Background(), "synthetic-tunnel")
	if r != nil || !errors.Is(e, sentinel) || w.calls != 1 {
		t.Fatal("transport", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w = &tunnelDeleteWire{t: t}
	r, e = tunnelDeleteClient(w).Tunnels.Delete(ctx, "synthetic-tunnel")
	if r != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("context", e)
	}
}

func TestTunnelsDeleteSetupAndBasic(t *testing.T) {
	w := &tunnelDeleteWire{t: t, code: 204, basic: true}
	c := sdk.NewClient(sdk.NewUsernamePasswordCredentialsProvider("synthetic-user", "synthetic-password"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
	r, e := c.Tunnels.Delete(context.Background(), "synthetic-tunnel")
	if e != nil || r == nil || w.calls != 1 || w.closed != 1 {
		t.Fatal("basic", e)
	}
	for _, endpoint := range []string{"http://example.invalid/", "https://example.invalid/api/?x=1"} {
		w = &tunnelDeleteWire{t: t}
		c = sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint(endpoint), sdk.WithHTTPClient(&http.Client{Transport: w}))
		r, e = c.Tunnels.Delete(context.Background(), "synthetic-tunnel")
		if e == nil || r != nil || w.calls != 0 {
			t.Fatal("endpoint refusal", endpoint, e)
		}
	}
	w = &tunnelDeleteWire{t: t}
	c = sdk.NewClient(sdk.NewTokenCredentialsProvider(""), sdk.WithHTTPClient(&http.Client{Transport: w}))
	r, e = c.Tunnels.Delete(context.Background(), "synthetic-tunnel")
	if e == nil || r != nil || w.calls != 0 {
		t.Fatal("credential refusal", e)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	w = &tunnelDeleteWire{t: t}
	r, e = tunnelDeleteClient(w).Tunnels.Delete(ctx, "synthetic-tunnel")
	if r != nil || !errors.Is(e, context.DeadlineExceeded) || w.calls > 1 {
		t.Fatal("deadline", e)
	}
}
