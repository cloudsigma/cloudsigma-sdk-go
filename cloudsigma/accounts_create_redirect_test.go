package cloudsigma_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type accountRedirectTransport struct {
	code          int
	location      string
	calls, closed atomic.Int64
}
type accountRedirectBody struct {
	io.Reader
	closed *atomic.Int64
}

func (b *accountRedirectBody) Close() error { b.closed.Add(1); return nil }
func (w *accountRedirectTransport) RoundTrip(q *http.Request) (*http.Response, error) {
	w.calls.Add(1)
	if q.Method != http.MethodPost {
		return nil, errors.New("unexpected follow-up GET")
	}
	return &http.Response{StatusCode: w.code, Header: http.Header{"Location": []string{w.location}}, Body: &accountRedirectBody{strings.NewReader("synthetic redirect body"), &w.closed}, Request: q}, nil
}
func TestAccountCreateRedirectRefusal(t *testing.T) {
	for _, code := range []int{301, 302, 303, 307, 308} {
		for i, location := range []string{
			"https://example.invalid/custom/api/2.0/accounts/action/?do=create",
			"https://example.invalid/custom/api/2.0/accounts/action/?do=other&secret=marker",
			"https://other.invalid/private?secret=marker",
			"//other.invalid/private?secret=marker",
			"?do=other&secret=marker",
			"/new-location?secret=marker",
		} {
			t.Run(fmt.Sprintf("%d/%d", code, i), func(t *testing.T) {
				w := &accountRedirectTransport{code: code, location: location}
				jar, err := cookiejar.New(nil)
				if err != nil {
					t.Fatal(err)
				}
				var callbacks atomic.Int64
				callback := func(_ *http.Request, _ []*http.Request) error {
					callbacks.Add(1)
					return errors.New("caller redirect policy")
				}
				client := &http.Client{Transport: w, Timeout: time.Minute, Jar: jar, CheckRedirect: callback}
				c := sdk.NewClient(accountCreateCredentials{}, sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(client))
				const count = 8
				var wg sync.WaitGroup
				for n := 0; n < count; n++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						value, response, err := c.CreateAccountRaw(context.Background(), sdk.AccountCreateRequest{Email: "synthetic@example.invalid"})
						if value != nil || response != nil || !errors.Is(err, sdk.ErrAccountCreateRedirect) || err.Error() != "account creation redirect refused; outcome unknown" {
							t.Errorf("redirect outcome %v %v %v", value, response, err)
						}
					}()
				}
				wg.Wait()
				if w.calls.Load() != count || w.closed.Load() != count || callbacks.Load() != 0 {
					t.Fatalf("dispatch/closure/callback %d %d %d", w.calls.Load(), w.closed.Load(), callbacks.Load())
				}
				if client.Transport != w || client.Jar != jar || client.Timeout != time.Minute || reflect.ValueOf(client.CheckRedirect).Pointer() != reflect.ValueOf(callback).Pointer() {
					t.Fatal("caller client mutated")
				}
				// The ordinary public send still consults the caller's redirect policy.
				req, err := c.NewRequest(http.MethodPost, "accounts/action/?do=create", sdk.AccountCreateRequest{})
				if err != nil {
					t.Fatal(err)
				}
				_, err = c.Do(context.Background(), req, nil)
				if err == nil {
					t.Fatal("ordinary redirect unexpectedly succeeded")
				}
				sameOrigin := i == 0 || i == 1 || i == 4 || i == 5
				if sameOrigin && callbacks.Load() != 1 {
					t.Fatal("ordinary redirect policy changed")
				}
				if w.calls.Load() != count+1 || w.closed.Load() != count+1 {
					t.Fatal("ordinary response ownership")
				}
			})
		}
	}
}
