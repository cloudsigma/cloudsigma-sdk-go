package cloudsigma_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
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

func TestLibraryDriveClonePreparseRedirectRefusal(t *testing.T) {
	for _, action := range []string{"nil", "empty", "named"} {
		for _, code := range []int{301, 302, 303, 307, 308} {
			for _, location := range []string{"https://USER-CANARY:PASSWORD-CANARY@example.invalid/target", "https://[::1]/target", "/SECRET-MUTATION-GAP/%zz", "/NUL\x00CANARY", "https://[::1", "/" + strings.Repeat("LOCATION-CANARY", 8192), "", "http://[SECRET", "1bad://SECRET", "/SECRET/\n", "https://example.invalid/target?secret=LOCATION", "https://other.invalid/target?secret=LOCATION", "/target?secret=LOCATION", "?do=other&secret=LOCATION", "../target?secret=LOCATION", "//other.invalid/target?secret=LOCATION", "//example.invalid/target?marker=LOCATION-CANARY"} {
				t.Run(fmt.Sprintf("%s/%d/location-bytes-%d", action, code, len(location)), func(t *testing.T) {
					var calls, closed, callbacks int32
					jar, _ := cookiejar.New(nil)
					u, _ := url.Parse("https://example.invalid/custom/api/2.0/")
					jar.SetCookies(u, []*http.Cookie{{Name: "sessionid", Value: "synthetic-session", Path: "/", Secure: true}})
					transport := resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
						atomic.AddInt32(&calls, 1)
						var b []byte
						if q.Body != nil {
							b, _ = io.ReadAll(q.Body)
							if q.Body != nil {
								_ = q.Body.Close()
							}
						}
						if q.Method != "POST" || q.URL.RequestURI() != librarydriveclonePreparsePath(action) || strings.TrimSpace(string(b)) != librarydriveclonePreparseBody(action) {
							t.Errorf("unexpected request: %s %s %q", q.Method, q.URL, b)
						}
						cookie, e := q.Cookie("sessionid")
						if e != nil || cookie.Value != "synthetic-session" || q.Header.Get("Authorization") != "" {
							t.Error("cookie-only credentials not preserved")
						}
						payload := "BODY-SECRET"
						status := code
						if q.URL.RequestURI() != librarydriveclonePreparsePath(action) {
							status = 202
							payload = `{ "uuid":"invented-id", "objects":[{"uuid":"invented-id"}] }`
						}
						return &http.Response{StatusCode: status, Header: http.Header{"Location": []string{location}, "Set-Cookie": []string{"sessionid=POISON; Path=/"}}, Body: &resourceRedirectBody{strings.NewReader(payload), &closed}, Request: q}, nil
					})
					callback := func(_ *http.Request, _ []*http.Request) error { atomic.AddInt32(&callbacks, 1); return nil }
					hc := &http.Client{Transport: transport, Jar: jar, Timeout: time.Second, CheckRedirect: callback}
					c := sdk.NewClient(logoutCookieCredentials{}, sdk.WithEndpoint(u.String()), sdk.WithHTTPClient(hc))
					var wg sync.WaitGroup
					for i := 0; i < 8; i++ {
						wg.Add(1)
						go func() {
							defer wg.Done()
							value, r, e := librarydriveclonePreparseCall(c, action)
							want := librarydriveclonePreparseError(action)
							if value || r != nil || e == nil || e.Error() != want || !errors.Is(e, librarydriveclonePreparseSentinel(action)) {
								t.Errorf("refusal value=%v response=%v error=%v", value, r, e)
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

func librarydriveclonePreparsePath(action string) string {
	return "/custom/api/2.0/libdrives/fixture/action/?do=clone"
}
func librarydriveclonePreparseBody(action string) string {
	switch action {
	case "nil":
		return ""
	case "empty":
		return `{"uuid":""}`
	default:
		return `{"name":"fixture","uuid":""}`
	}
}
func librarydriveclonePayload(action string) *sdk.LibraryDriveCloneRequest {
	if action == "nil" {
		return nil
	}
	if action == "empty" {
		return &sdk.LibraryDriveCloneRequest{LibraryDrive: &sdk.LibraryDrive{}}
	}
	return &sdk.LibraryDriveCloneRequest{LibraryDrive: &sdk.LibraryDrive{Name: "fixture"}}
}
func librarydriveclonePreparseSentinel(action string) error { return sdk.ErrLibraryDriveCloneRedirect }
func librarydriveclonePreparseError(action string) string {
	return "library drive clone redirect refused; outcome unknown"
}
func librarydriveclonePreparseCallContext(ctx context.Context, c *sdk.Client, action string) (bool, *sdk.Response, error) {
	v, r, e := c.LibraryDrives.Clone(ctx, "fixture", librarydriveclonePayload(action))
	return v != nil, r, e
}
func librarydriveclonePreparseCall(c *sdk.Client, action string) (bool, *sdk.Response, error) {
	return librarydriveclonePreparseCallContext(context.Background(), c, action)
}
func TestLibraryDriveClonePreparseTimeoutCancellation(t *testing.T) {
	for _, action := range []string{"nil", "empty", "named"} {
		for _, mode := range []string{"timeout", "cancel"} {
			t.Run(action+"/"+mode, func(t *testing.T) {
				var calls int32
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				hc := &http.Client{Timeout: 10 * time.Millisecond, Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
					atomic.AddInt32(&calls, 1)
					if q.Body != nil {
						_ = q.Body.Close()
					}
					if mode == "cancel" {
						cancel()
					}
					<-q.Context().Done()
					return nil, q.Context().Err()
				})}
				c := sdk.NewClient(logoutCookieCredentials{}, sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(hc))
				v, r, e := librarydriveclonePreparseCallContext(ctx, c, action)
				if v || r != nil || calls != 1 {
					t.Fatal("ownership")
				}
				if mode == "cancel" {
					if !errors.Is(e, context.Canceled) {
						t.Fatal("cancellation lost")
					}
				} else {
					var ne net.Error
					if !errors.As(e, &ne) || !ne.Timeout() {
						t.Fatal("timeout lost")
					}
				}
			})
		}
	}
}
