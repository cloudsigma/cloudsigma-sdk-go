package cloudsigma_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestSubscriptionCreatePreparseCookieAndEmptyLocation(t *testing.T) {
	for _, action := range []string{"single", "renew"} {
		for _, code := range []int{200, 201, 202, 301, 302, 303, 307, 308, 400} {
			for _, loc := range []string{"", "http://[SECRET-LOCATION-CANARY", "/%zz"} {
				t.Run(fmt.Sprintf("%s/%d/%q", action, code, loc), func(t *testing.T) {
					var calls, closed, callbacks int32
					jar, _ := cookiejar.New(nil)
					u, _ := url.Parse("https://example.invalid/custom/api/2.0/")
					jar.SetCookies(u, []*http.Cookie{{Name: "sessionid", Value: "before", Path: "/"}})
					hc := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { atomic.AddInt32(&callbacks, 1); return nil }, Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
						atomic.AddInt32(&calls, 1)
						if q.Body != nil {
							_ = q.Body.Close()
						}
						if q.Method != "POST" {
							t.Error("not POST")
						}
						h := http.Header{"Set-Cookie": []string{"sessionid=after; Path=/"}}
						if loc != "" {
							h.Set("Location", loc)
						}
						payload := `{"uuid":"ack","objects":[{"uuid":"ack"}]}`
						return &http.Response{StatusCode: code, Header: h, Body: &resourceRedirectBody{strings.NewReader(payload), &closed}, Request: q}, nil
					})}
					c := sdk.NewClient(logoutCookieCredentials{}, sdk.WithEndpoint(u.String()), sdk.WithHTTPClient(hc))
					v, r, e := subscriptioncreatePreparseCall(c, action)
					redirect := code == 301 || code == 302 || code == 303 || code == 307 || code == 308
					wantCookie := "after"
					if redirect {
						wantCookie = "before"
						if v || r != nil || e == nil || e.Error() != subscriptioncreatePreparseError(action) {
							t.Fatalf("unsafe redirect %v %v %v", v, r, e)
						}
					} else if code < 300 {
						if !v || r == nil || e != nil || r.StatusCode != code {
							t.Fatalf("normal response changed %v %v %v", v, r, e)
						}
					} else if e == nil || r == nil {
						t.Fatal("HTTP error lost")
					}
					cookies := jar.Cookies(u)
					if len(cookies) != 1 || cookies[0].Value != wantCookie {
						t.Fatalf("cookie policy %v want %s", cookies, wantCookie)
					}
					if calls != 1 || closed != 1 || callbacks != 0 || hc.Jar != jar {
						t.Fatalf("ownership %d %d %d", calls, closed, callbacks)
					}
				})
			}
		}
	}
}

func TestSubscriptionCreatePreparseTransportErrorOwnership(t *testing.T) {
	for _, action := range []string{"single", "renew"} {
		for _, withResponse := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", action, withResponse), func(t *testing.T) {
				var closed int32
				hc := &http.Client{Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
					if q.Body != nil {
						_ = q.Body.Close()
					}
					if !withResponse {
						return nil, context.Canceled
					}
					return &http.Response{StatusCode: 307, Header: http.Header{"Location": []string{"/%zz-SECRET"}}, Body: &resourceRedirectBody{strings.NewReader("SECRET"), &closed}}, context.Canceled
				})}
				c := sdk.NewClient(logoutCookieCredentials{}, sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(hc))
				v, r, e := subscriptioncreatePreparseCall(c, action)
				if v || r != nil || !errors.Is(e, context.Canceled) {
					t.Fatalf("transport error changed %v %v %v", v, r, e)
				}
				want := int32(0)
				if withResponse {
					want = 1
				}
				if closed != want {
					t.Fatalf("close %d want %d", closed, want)
				}
			})
		}
	}
}
