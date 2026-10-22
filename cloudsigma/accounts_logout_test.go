package cloudsigma_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type accountLogoutWire struct {
	t                   *testing.T
	body                string
	code, calls, closed int
	err                 error
	basic               bool
	cookie              bool
}
type accountLogoutBody struct {
	*strings.Reader
	w *accountLogoutWire
}

func (b *accountLogoutBody) Close() error { b.w.closed++; return nil }
func (w *accountLogoutWire) RoundTrip(q *http.Request) (*http.Response, error) {
	w.calls++
	if q.Method != "POST" || q.URL.RequestURI() != "/custom/api/2.0/accounts/action/?do=logout" {
		w.t.Fatalf("route %s %s", q.Method, q.URL)
	}
	if w.cookie {
		c, e := q.Cookie("sessionid")
		if e != nil || c.Value != "synthetic-session" || q.Header.Get("Authorization") != "" {
			w.t.Fatal("caller cookie wiring")
		}
	} else if w.basic {
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
	return &http.Response{StatusCode: w.code, Header: make(http.Header), Body: &accountLogoutBody{strings.NewReader(w.body), w}, Request: q}, nil
}
func accountLogoutClient(w *accountLogoutWire) *sdk.Client {
	return sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
}
func TestAccountLogoutAcknowledgment(t *testing.T) {
	w := &accountLogoutWire{t: t, code: 200}
	r, e := accountLogoutClient(w).Logout(context.Background())
	if e != nil || r == nil || r.StatusCode != 200 || r.Meta != nil || w.calls != 1 || w.closed != 1 {
		t.Fatalf("acknowledgment %v %v %+v", r, e, w)
	}
	// One request; no login, cookie acquisition or durable session is inferred.
}
func TestAccountLogoutErrors(t *testing.T) {
	for _, code := range []int{400, 401, 403, 404, 409, 429, 500} {
		w := &accountLogoutWire{t: t, code: code, body: `[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`}
		r, e := accountLogoutClient(w).Logout(context.Background())
		var api *sdk.ErrorResponse
		if r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0].Message != "synthetic refusal" || w.calls != 1 || w.closed != 1 {
			t.Fatalf("HTTP%d: %v", code, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	w := &accountLogoutWire{t: t, err: sentinel}
	r, e := accountLogoutClient(w).Logout(context.Background())
	if r != nil || !errors.Is(e, sentinel) || w.calls != 1 {
		t.Fatal("transport", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w = &accountLogoutWire{t: t}
	r, e = accountLogoutClient(w).Logout(ctx)
	if r != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("context", e)
	}
}

func TestAccountLogoutSetupAndBasic(t *testing.T) {
	w := &accountLogoutWire{t: t, code: 200, basic: true}
	c := sdk.NewClient(sdk.NewUsernamePasswordCredentialsProvider("synthetic-user", "synthetic-password"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
	r, e := c.Logout(context.Background())
	if e != nil || r == nil || w.calls != 1 || w.closed != 1 {
		t.Fatal("basic", e)
	}
	for _, endpoint := range []string{"http://example.invalid/", "https://example.invalid/api/?x=1"} {
		w = &accountLogoutWire{t: t}
		c = sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint(endpoint), sdk.WithHTTPClient(&http.Client{Transport: w}))
		r, e = c.Logout(context.Background())
		if e == nil || r != nil || w.calls != 0 {
			t.Fatal("endpoint refusal", endpoint, e)
		}
	}
	w = &accountLogoutWire{t: t}
	c = sdk.NewClient(sdk.NewTokenCredentialsProvider(""), sdk.WithHTTPClient(&http.Client{Transport: w}))
	r, e = c.Logout(context.Background())
	if e == nil || r != nil || w.calls != 0 {
		t.Fatal("credential refusal", e)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	w = &accountLogoutWire{t: t}
	r, e = accountLogoutClient(w).Logout(ctx)
	if r != nil || !errors.Is(e, context.DeadlineExceeded) || w.calls > 1 {
		t.Fatal("deadline", e)
	}
}

// A caller-provided credential source and jar exercise existing public wiring,
// not login, CSRF acquisition, or real server-side session invalidation.
type logoutCookieCredentials struct{}

func (logoutCookieCredentials) Retrieve() (sdk.Credentials, error) { return sdk.Credentials{}, nil }
func TestAccountLogoutCallerCookie(t *testing.T) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse("https://example.invalid/custom/api/2.0/")
	if err != nil {
		t.Fatal(err)
	}
	jar.SetCookies(u, []*http.Cookie{{Name: "sessionid", Value: "synthetic-session", Path: "/", Secure: true}})
	w := &accountLogoutWire{t: t, code: 200, cookie: true}
	hc := &http.Client{Transport: w, Jar: jar}
	c := sdk.NewClient(logoutCookieCredentials{}, sdk.WithEndpoint(u.String()), sdk.WithHTTPClient(hc))
	r, e := c.Logout(context.Background())
	if e != nil || r == nil || r.StatusCode != 200 || w.calls != 1 || w.closed != 1 {
		t.Fatalf("cookie ack %v %v", r, e)
	}
	cookies := jar.Cookies(u)
	if hc.Jar != jar || len(cookies) != 1 || cookies[0].Value != "synthetic-session" {
		t.Fatal("caller jar changed without server cookie instruction")
	}
}
