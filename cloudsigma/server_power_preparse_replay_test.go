package cloudsigma_test

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// This identical oracle is also run with the baseline production file.
// The fixture commits the first POST and would return an invented identity on
// follow-up. Refusal cannot establish whether that initial commit succeeded.
func TestPowerPreparseReplayAndPayload(t *testing.T) {
	for _, action := range []string{"start", "stop", "shutdown", "avoid"} {
		for _, code := range []int{301, 302, 303, 307, 308} {
			for _, options := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%d/options=%v", action, code, options), func(t *testing.T) {
					var calls, closed int32
					var methods, bodies []string
					transport := resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
						n := atomic.AddInt32(&calls, 1)
						wantAuth := "Bearer synthetic-secret"
						if options {
							wantAuth = "Bearer synthetic-other"
						}
						if q.Header.Get("Authorization") != wantAuth {
							t.Error("token authentication changed")
						}
						body := ""
						if q.Body != nil {
							b, _ := io.ReadAll(q.Body)
							_ = q.Body.Close()
							body = strings.TrimSpace(string(b))
						}
						methods = append(methods, q.Method)
						bodies = append(bodies, body)
						if n == 1 && q.URL.RequestURI() != powerPreparsePath(action) {
							t.Error("initial route changed")
						}
						if n == 1 {
							return &http.Response{StatusCode: code, Header: http.Header{"Location": []string{"/committed?SECRET=location"}}, Body: &resourceRedirectBody{Reader: strings.NewReader("BODY-SECRET"), closed: &closed}, Request: q}, nil
						}
						return &http.Response{StatusCode: 202, Header: make(http.Header), Body: &resourceRedirectBody{Reader: strings.NewReader(`{"uuid":"invented","objects":[{"uuid":"invented"}]}`), closed: &closed}, Request: q}, nil
					})
					credentials := sdk.NewTokenCredentialsProvider("synthetic-secret")
					if options {
						credentials = sdk.NewTokenCredentialsProvider("synthetic-other")
					}
					c := sdk.NewClient(credentials, sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: transport}))
					var value bool
					var r *sdk.Response
					var e error
					wantBody := powerPreparseBody(action)
					value, r, e = powerPreparseCall(c, action)
					t.Logf("actual calls=%d methods=%v bodies=%q closed=%d value=%v responsePresent=%v error=%v", calls, methods, bodies, closed, value, r != nil, e)
					if len(bodies) == 0 || bodies[0] != wantBody {
						t.Error("initial optional payload changed")
					}
					if calls != 1 || closed != 1 || value || r != nil || e != powerPreparseSentinel(action) {
						t.Error("redirect admitted or unknown outcome not refused")
					}
				})
			}
		}
	}
}
