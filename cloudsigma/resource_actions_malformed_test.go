package cloudsigma_test

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestIndependentMalformedLocationPrivacy(t *testing.T) {
	for _, action := range []string{"clone", "create", "start", "stop"} {
		for _, code := range []int{301, 302, 303, 307, 308} {
			for name, loc := range map[string]string{"invalidEscape": "/SECRET-LOCATION-CANARY/%invalid", "invalidScheme": "1bad://SECRET-LOCATION-CANARY", "userinfo": "https://user:SECRET-LOCATION-CANARY@example.invalid/%zz", "control": "/SECRET-LOCATION-CANARY/\n", "oversized": "/" + strings.Repeat("x", 8192) + "SECRET-LOCATION-CANARY/%zz", "validUserinfo": "https://user:SECRET-LOCATION-CANARY@example.invalid/ok"} {
				t.Run(fmt.Sprintf("%s/%d/%s", action, code, name), func(t *testing.T) {
					var calls, closed, callbacks int32
					hc := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { atomic.AddInt32(&callbacks, 1); return nil }, Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
						atomic.AddInt32(&calls, 1)
						q.Body.Close()
						return &http.Response{StatusCode: code, Header: http.Header{"Location": []string{loc}}, Body: &resourceRedirectBody{strings.NewReader("BODY-CANARY"), &closed}, Request: q}, nil
					})}
					c := sdk.NewClient(logoutCookieCredentials{}, sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(hc))
					v, r, e := resourceActionCall(c, action)
					want := map[string]error{"clone": sdk.ErrSnapshotCloneRedirect, "create": sdk.ErrBServerCreateRedirect, "start": sdk.ErrBServerStartRedirect, "stop": sdk.ErrBServerStopRedirect}[action]
					if calls != 1 || closed != 1 || callbacks != 0 || v || r != nil {
						t.Errorf("dispatch/body/ack: %d %d %d %v %v", calls, closed, callbacks, v, r)
					}
					if !errors.Is(e, want) || e.Error() != want.Error() {
						t.Errorf("unsafe refusal: sentinel=%v leakedCanary=%v errorType=%T", errors.Is(e, want), e != nil && strings.Contains(e.Error(), "SECRET-LOCATION-CANARY"), e)
					}
				})
			}
		}
	}
}
