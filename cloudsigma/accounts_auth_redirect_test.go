package cloudsigma_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type authRedirectTransport func(*http.Request) (*http.Response, error)

func (f authRedirectTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type authRedirectBody struct {
	io.Reader
	closed *int32
}

func (b *authRedirectBody) Close() error { atomic.AddInt32(b.closed, 1); return nil }

func TestAuthActionRedirectRefusal(t *testing.T) {
	for _, action := range []string{"check_login", "logout"} {
		for _, code := range []int{301, 302, 303, 307, 308} {
			for _, location := range []string{"https://example.invalid/target?secret=LOCATION", "https://other.invalid/target?secret=LOCATION", "/target?secret=LOCATION", "?do=other&secret=LOCATION", "../target?secret=LOCATION", "//other.invalid/target?secret=LOCATION"} {
				t.Run(fmt.Sprintf("%s/%d/%s", action, code, location), func(t *testing.T) {
					var calls, closed, callbacks int32
					jar, _ := cookiejar.New(nil)
					u, _ := url.Parse("https://example.invalid/custom/api/2.0/")
					jar.SetCookies(u, []*http.Cookie{{Name: "sessionid", Value: "synthetic-session", Path: "/", Secure: true}})
					transport := authRedirectTransport(func(q *http.Request) (*http.Response, error) {
						atomic.AddInt32(&calls, 1)
						var b []byte
						if q.Body != nil {
							b, _ = io.ReadAll(q.Body)
							_ = q.Body.Close()
						}
						if q.Method != "POST" || q.URL.RequestURI() != "/custom/api/2.0/accounts/action/?do="+action || strings.TrimSpace(string(b)) != "{}" {
							t.Errorf("unexpected request: %s %s %q", q.Method, q.URL, b)
						}
						cookie, e := q.Cookie("sessionid")
						if e != nil || cookie.Value != "synthetic-session" || q.Header.Get("Authorization") != "" {
							t.Error("cookie-only credentials not preserved")
						}
						status := code
						if q.URL.RequestURI() != "/custom/api/2.0/accounts/action/?do="+action {
							status = 200
						}
						return &http.Response{StatusCode: status, Header: http.Header{"Location": []string{location}}, Body: &authRedirectBody{strings.NewReader("BODY-SECRET"), &closed}, Request: q}, nil
					})
					callback := func(_ *http.Request, _ []*http.Request) error { atomic.AddInt32(&callbacks, 1); return nil }
					hc := &http.Client{Transport: transport, Jar: jar, Timeout: time.Second, CheckRedirect: callback}
					c := sdk.NewClient(logoutCookieCredentials{}, sdk.WithEndpoint(u.String()), sdk.WithHTTPClient(hc))
					var wg sync.WaitGroup
					for i := 0; i < 8; i++ {
						wg.Add(1)
						go func() {
							defer wg.Done()
							var r *sdk.Response
							var e error
							if action == "logout" {
								r, e = c.Logout(context.Background())
							} else {
								r, e = c.CheckLogin(context.Background())
							}
							// String oracle also compiles on the unpatched baseline, proving actual behavior.
							want := "login check redirect refused; outcome unknown"
							if action == "logout" {
								want = "logout redirect refused; outcome unknown"
							}
							if r != nil || e == nil || e.Error() != want {
								t.Errorf("refusal response=%v error=%v", r, e)
							}
							var ue *url.Error
							if errors.As(e, &ue) {
								t.Error("URL error leaked")
							}
						}()
					}
					wg.Wait()
					if calls != 8 || closed != 8 || callbacks != 0 {
						t.Errorf("calls=%d closed=%d callback=%d want 8/8/0", calls, closed, callbacks)
					}
					if reflect.ValueOf(hc.Transport).Pointer() != reflect.ValueOf(transport).Pointer() || hc.Jar != jar || hc.Timeout != time.Second || reflect.ValueOf(hc.CheckRedirect).Pointer() != reflect.ValueOf(callback).Pointer() {
						t.Error("shared HTTP configuration changed")
					}
					if got := jar.Cookies(u); len(got) != 1 || got[0].Value != "synthetic-session" {
						t.Error("caller jar changed")
					}
				})
			}
		}
	}
}

func TestAuthActionLegacyDoRedirect(t *testing.T) {
	var calls, closed, callbacks int32
	hc := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { callbacks++; return nil }}
	hc.Transport = authRedirectTransport(func(q *http.Request) (*http.Response, error) {
		calls++
		code := 200
		h := http.Header{}
		if calls == 1 {
			code = 307
			h.Set("Location", "/next")
		}
		return &http.Response{StatusCode: code, Header: h, Body: &authRedirectBody{strings.NewReader(""), &closed}, Request: q}, nil
	})
	c := sdk.NewClient(logoutCookieCredentials{}, sdk.WithEndpoint("https://example.invalid/"), sdk.WithHTTPClient(hc))
	q, e := c.NewRequest("POST", "ordinary", struct{}{})
	if e != nil {
		t.Fatal(e)
	}
	r, e := c.Do(context.Background(), q, nil)
	if e != nil || r == nil || calls != 2 || closed != 2 || callbacks != 1 {
		t.Fatalf("legacy %v %v %d/%d/%d", r, e, calls, closed, callbacks)
	}
}

func TestAuthActionSentinelsAndTimeout(t *testing.T) {
	for _, logout := range []bool{false, true} {
		for _, mode := range []string{"redirect", "timeout", "empty401", "ack"} {
			t.Run(fmt.Sprintf("logout=%t/%s", logout, mode), func(t *testing.T) {
				var calls, closed int32
				hc := &http.Client{Timeout: 10 * time.Millisecond}
				hc.Transport = authRedirectTransport(func(q *http.Request) (*http.Response, error) {
					atomic.AddInt32(&calls, 1)
					if mode == "timeout" {
						<-q.Context().Done()
						return nil, q.Context().Err()
					}
					code := 200
					h := http.Header{}
					if mode == "redirect" {
						code = 307
						h.Set("Location", "/target?private=synthetic-location")
					}
					if mode == "empty401" {
						code = 401
					}
					return &http.Response{StatusCode: code, Header: h, Body: &authRedirectBody{strings.NewReader(""), &closed}, Request: q}, nil
				})
				c := sdk.NewClient(logoutCookieCredentials{}, sdk.WithEndpoint("https://example.invalid/"), sdk.WithHTTPClient(hc))
				var r *sdk.Response
				var e error
				if logout {
					r, e = c.Logout(context.Background())
				} else {
					r, e = c.CheckLogin(context.Background())
				}
				switch mode {
				case "redirect":
					want := sdk.ErrCheckLoginRedirect
					if logout {
						want = sdk.ErrLogoutRedirect
					}
					if r != nil || e != want || !errors.Is(fmt.Errorf("wrapped: %w", e), want) || errors.Is(e, sdk.ErrAccountCreateRedirect) {
						t.Fatalf("sentinel %v %v", r, e)
					}
				case "timeout":
					var timeout interface{ Timeout() bool }
					if r != nil || !errors.As(e, &timeout) || !timeout.Timeout() {
						t.Fatalf("timeout not preserved: %v", e)
					}
				case "empty401":
					var api *sdk.ErrorResponse
					if r == nil || r.StatusCode != 401 || !errors.As(e, &api) {
						t.Fatalf("empty HTTP error %v %v", r, e)
					}
				case "ack":
					if r == nil || e != nil {
						t.Fatalf("ack %v %v", r, e)
					}
				}
				if calls != 1 || (mode != "timeout" && closed != 1) {
					t.Fatalf("counts %d/%d", calls, closed)
				}
			})
		}
	}
}
