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

type resourceRedirectTransport func(*http.Request) (*http.Response, error)

func (f resourceRedirectTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type resourceRedirectBody struct {
	io.Reader
	closed *int32
}

func (b *resourceRedirectBody) Close() error { atomic.AddInt32(b.closed, 1); return nil }

func TestResourceActionRedirectRefusal(t *testing.T) {
	for _, action := range []string{"clone", "create", "start", "stop"} {
		for _, code := range []int{301, 302, 303, 307, 308} {
			for _, location := range []string{"https://example.invalid/target?secret=LOCATION", "https://other.invalid/target?secret=LOCATION", "/target?secret=LOCATION", "?do=other&secret=LOCATION", "../target?secret=LOCATION", "//other.invalid/target?secret=LOCATION", "//example.invalid/target?marker=LOCATION-CANARY"} {
				t.Run(fmt.Sprintf("%s/%d/%s", action, code, location), func(t *testing.T) {
					var calls, closed, callbacks int32
					jar, _ := cookiejar.New(nil)
					u, _ := url.Parse("https://example.invalid/custom/api/2.0/")
					jar.SetCookies(u, []*http.Cookie{{Name: "sessionid", Value: "synthetic-session", Path: "/", Secure: true}})
					transport := resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
						atomic.AddInt32(&calls, 1)
						var b []byte
						if q.Body != nil {
							b, _ = io.ReadAll(q.Body)
							_ = q.Body.Close()
						}
						if q.Method != "POST" || q.URL.RequestURI() != resourceActionPath(action) || strings.TrimSpace(string(b)) != resourceActionBody(action) {
							t.Errorf("unexpected request: %s %s %q", q.Method, q.URL, b)
						}
						cookie, e := q.Cookie("sessionid")
						if e != nil || cookie.Value != "synthetic-session" || q.Header.Get("Authorization") != "" {
							t.Error("cookie-only credentials not preserved")
						}
						payload := "BODY-SECRET"
						status := code
						if q.URL.RequestURI() != resourceActionPath(action) {
							status = 202
							payload = `{ "uuid":"invented-id", "objects":[{"uuid":"invented-id"}] }`
						}
						return &http.Response{StatusCode: status, Header: http.Header{"Location": []string{location}}, Body: &resourceRedirectBody{strings.NewReader(payload), &closed}, Request: q}, nil
					})
					callback := func(_ *http.Request, _ []*http.Request) error { atomic.AddInt32(&callbacks, 1); return nil }
					hc := &http.Client{Transport: transport, Jar: jar, Timeout: time.Second, CheckRedirect: callback}
					c := sdk.NewClient(logoutCookieCredentials{}, sdk.WithEndpoint(u.String()), sdk.WithHTTPClient(hc))
					var wg sync.WaitGroup
					for i := 0; i < 8; i++ {
						wg.Add(1)
						go func() {
							defer wg.Done()
							value, r, e := resourceActionCall(c, action)
							want := resourceActionError(action)
							if value || r != nil || e == nil || e.Error() != want {
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

func resourceActionPath(action string) string {
	if action == "create" {
		return "/custom/api/2.0/bservers/"
	}
	resource := "bservers"
	if action == "clone" {
		resource = "snapshots"
	}
	return "/custom/api/2.0/" + resource + "/synthetic-id/action/?do=" + action
}
func resourceActionBody(action string) string {
	if action == "create" {
		return `{"objects":[{"cpu":1,"mem":2,"name":"synthetic","vnc_password":"BODY-CREDENTIAL-CANARY"}]}`
	}
	return "{}"
}
func resourceActionError(action string) string {
	if action == "clone" {
		return "snapshot clone redirect refused; outcome unknown"
	}
	return "backfilling " + action + " redirect refused; outcome unknown"
}
func resourceActionCall(c *sdk.Client, action string) (bool, *sdk.Response, error) {
	switch action {
	case "clone":
		v, r, e := c.Snapshots.CloneRaw(context.Background(), "synthetic-id")
		return v != nil, r, e
	case "create":
		v, r, e := c.BServers.CreateMinimalRaw(context.Background(), []sdk.BServerMinimalCreateRequest{{CPU: 1, Mem: 2, Name: "synthetic", VNCPassword: "BODY-CREDENTIAL-CANARY"}})
		return v != nil, r, e
	case "start":
		v, r, e := c.BServers.Start(context.Background(), "synthetic-id")
		return v != nil, r, e
	default:
		v, r, e := c.BServers.Stop(context.Background(), "synthetic-id")
		return v != nil, r, e
	}
}

func TestResourceActionSentinelsTimeoutAndAcknowledgment(t *testing.T) {
	for _, action := range []string{"clone", "create", "start", "stop"} {
		for _, mode := range []string{"redirect", "timeout", "ack"} {
			t.Run(action+"/"+mode, func(t *testing.T) {
				var calls, closed int32
				hc := &http.Client{Timeout: 10 * time.Millisecond}
				hc.Transport = resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
					atomic.AddInt32(&calls, 1)
					b, _ := io.ReadAll(q.Body)
					_ = q.Body.Close()
					if q.Method != "POST" || q.URL.RequestURI() != resourceActionPath(action) || strings.TrimSpace(string(b)) != resourceActionBody(action) {
						t.Error("initial request changed")
					}
					if mode == "timeout" {
						<-q.Context().Done()
						return nil, q.Context().Err()
					}
					code := 202
					payload := `{"uuid":"ack-id","nested":{"number":9007199254740993}}`
					h := http.Header{}
					if action == "create" {
						code = 201
						payload = `{"objects":[{"uuid":"ack-id"}],"meta":{"total_count":1}}`
					}
					if mode == "redirect" {
						code = 307
						h.Set("Location", "/target?secret=URL-CANARY")
						payload = "BODY-CREDENTIAL-CANARY"
					}
					return &http.Response{StatusCode: code, Header: h, Body: &resourceRedirectBody{strings.NewReader(payload), &closed}, Request: q}, nil
				})
				c := sdk.NewClient(logoutCookieCredentials{}, sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(hc))
				value, r, e := resourceActionCall(c, action)
				switch mode {
				case "redirect":
					want := map[string]error{"clone": sdk.ErrSnapshotCloneRedirect, "create": sdk.ErrBServerCreateRedirect, "start": sdk.ErrBServerStartRedirect, "stop": sdk.ErrBServerStopRedirect}[action]
					if value || r != nil || e != want || !errors.Is(fmt.Errorf("wrapped: %w", e), want) || errors.Is(e, sdk.ErrAccountCreateRedirect) {
						t.Fatalf("sentinel %v %v %v", value, r, e)
					}
				case "timeout":
					var ne interface{ Timeout() bool }
					if value || r != nil || !errors.As(e, &ne) || !ne.Timeout() {
						t.Fatalf("timeout %v %v %v", value, r, e)
					}
				case "ack":
					want := 202
					if action == "create" {
						want = 201
					}
					if !value || r == nil || r.StatusCode != want || e != nil {
						t.Fatalf("ack %v %v %v", value, r, e)
					}
				}
				if calls != 1 || (mode != "timeout" && closed != 1) {
					t.Fatalf("counts %d/%d", calls, closed)
				}
			})
		}
	}
}
