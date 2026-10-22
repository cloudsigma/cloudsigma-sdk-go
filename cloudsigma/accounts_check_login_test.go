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

type accountCheckLoginWire struct {
	t                   *testing.T
	body                string
	code, calls, closed int
	err                 error
	basic               bool
}
type accountCheckLoginBody struct {
	*strings.Reader
	w *accountCheckLoginWire
}

func (b *accountCheckLoginBody) Close() error { b.w.closed++; return nil }
func (w *accountCheckLoginWire) RoundTrip(q *http.Request) (*http.Response, error) {
	w.calls++
	if q.Method != "POST" || q.URL.RequestURI() != "/custom/api/2.0/accounts/action/?do=check_login" {
		w.t.Fatalf("route %s %s", q.Method, q.URL)
	}
	if w.basic {
		u, p, ok := q.BasicAuth()
		if !ok || u != "synthetic-user" || p != "synthetic-password" {
			w.t.Fatal("basic auth")
		}
	} else if q.Header.Get("Authorization") != "Bearer synthetic" {
		w.t.Fatal("authentication")
	}
	if q.Header.Get("Content-Type") != "application/json" || q.Header.Get("Accept") != "application/json" {
		w.t.Fatal("headers")
	}
	if q.Body == nil {
		w.t.Fatal("missing body")
	}
	if q.Body != nil {
		b, e := io.ReadAll(q.Body)
		if e != nil || strings.TrimSpace(string(b)) != "{}" {
			w.t.Fatal("request body")
		}
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if w.err != nil {
		return nil, w.err
	}
	return &http.Response{StatusCode: w.code, Header: make(http.Header), Body: &accountCheckLoginBody{strings.NewReader(w.body), w}, Request: q}, nil
}
func accountCheckLoginClient(w *accountCheckLoginWire) *sdk.Client {
	return sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
}
func TestAccountCheckLoginAcknowledgment(t *testing.T) {
	w := &accountCheckLoginWire{t: t, code: 200}
	r, e := accountCheckLoginClient(w).CheckLogin(context.Background())
	if e != nil || r == nil || r.StatusCode != 200 || r.Meta != nil || w.calls != 1 || w.closed != 1 {
		t.Fatalf("acknowledgment %v %v %+v", r, e, w)
	}
	// One request; no login, cookie acquisition or durable session is inferred.
}
func TestAccountCheckLoginErrors(t *testing.T) {
	for _, code := range []int{400, 401, 403, 404, 409, 429, 500} {
		w := &accountCheckLoginWire{t: t, code: code, body: `[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`}
		r, e := accountCheckLoginClient(w).CheckLogin(context.Background())
		var api *sdk.ErrorResponse
		if r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0].Message != "synthetic refusal" || w.calls != 1 || w.closed != 1 {
			t.Fatalf("HTTP%d: %v", code, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	w := &accountCheckLoginWire{t: t, err: sentinel}
	r, e := accountCheckLoginClient(w).CheckLogin(context.Background())
	if r != nil || !errors.Is(e, sentinel) || w.calls != 1 {
		t.Fatal("transport", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w = &accountCheckLoginWire{t: t}
	r, e = accountCheckLoginClient(w).CheckLogin(ctx)
	if r != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("context", e)
	}
}

func TestAccountCheckLoginSetupAndBasic(t *testing.T) {
	w := &accountCheckLoginWire{t: t, code: 200, basic: true}
	c := sdk.NewClient(sdk.NewUsernamePasswordCredentialsProvider("synthetic-user", "synthetic-password"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
	r, e := c.CheckLogin(context.Background())
	if e != nil || r == nil || w.calls != 1 || w.closed != 1 {
		t.Fatal("basic", e)
	}
	for _, endpoint := range []string{"http://example.invalid/", "https://example.invalid/api/?x=1"} {
		w = &accountCheckLoginWire{t: t}
		c = sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint(endpoint), sdk.WithHTTPClient(&http.Client{Transport: w}))
		r, e = c.CheckLogin(context.Background())
		if e == nil || r != nil || w.calls != 0 {
			t.Fatal("endpoint refusal", endpoint, e)
		}
	}
	w = &accountCheckLoginWire{t: t}
	c = sdk.NewClient(sdk.NewTokenCredentialsProvider(""), sdk.WithHTTPClient(&http.Client{Transport: w}))
	r, e = c.CheckLogin(context.Background())
	if e == nil || r != nil || w.calls != 0 {
		t.Fatal("credential refusal", e)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	w = &accountCheckLoginWire{t: t}
	r, e = accountCheckLoginClient(w).CheckLogin(ctx)
	if r != nil || !errors.Is(e, context.DeadlineExceeded) || w.calls > 1 {
		t.Fatal("deadline", e)
	}
}
