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

func TestResizeDefinitionPreparseRedirectRefusal(t *testing.T) {
	for _, action := range []string{"drive"} {
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
						if q.Method != "POST" || q.URL.RequestURI() != resizeDefinitionPreparsePath(action) || strings.TrimSpace(string(b)) != resizeDefinitionPreparseBody(action) {
							t.Errorf("unexpected request: %s %s %q", q.Method, q.URL, b)
						}
						cookie, e := q.Cookie("sessionid")
						if e != nil || cookie.Value != "synthetic-session" || q.Header.Get("Authorization") != "" {
							t.Error("cookie-only credentials not preserved")
						}
						payload := "BODY-SECRET"
						status := code
						if q.URL.RequestURI() != resizeDefinitionPreparsePath(action) {
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
							value, r, e := resizeDefinitionPreparseCall(c, action)
							want := resizeDefinitionPreparseError(action)
							if value || r != nil || e == nil || e.Error() != want || !errors.Is(e, resizeDefinitionPreparseSentinel(action)) {
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

func resizeDefinitionPreparsePath(action string) string {
	return "/custom/api/2.0/drives/drive-id/action/?do=resize"
}
func resizeDefinitionPreparseBody(action string) string { return `{"name":"resize","size":4096}` }
func resizeDefinitionPreparseSentinel(action string) error {
	return sdk.ErrDriveResizeDefinitionRedirect
}
func resizeDefinitionPreparseError(action string) string {
	return "drive resize definition redirect refused; outcome unknown"
}
func resizeDefinitionPreparseCallContext(ctx context.Context, c *sdk.Client, action string) (bool, *sdk.Response, error) {
	v, r, e := c.Drives.ResizeDefinition(ctx, "drive-id", sdk.DriveDefinition{"name": []byte(`"resize"`), "size": []byte(`4096`)})
	return v != nil, r, e
}
func resizeDefinitionPreparseCall(c *sdk.Client, action string) (bool, *sdk.Response, error) {
	return resizeDefinitionPreparseCallContext(context.Background(), c, action)
}
func TestResizeDefinitionPreparseTimeoutCancellation(t *testing.T) {
	for _, action := range []string{"drive"} {
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
				v, r, e := resizeDefinitionPreparseCallContext(ctx, c, action)
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
