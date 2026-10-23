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
func TestACLCreateDefinitionPreparseReplayAndPayload(t *testing.T) {
	for _, action := range []string{"empty", "grantees"} {
		for _, code := range []int{301, 302, 303, 307, 308} {
			for _, repeat := range []int{1, 2} {
				t.Run(fmt.Sprintf("%s/%d/repeat=%v", action, code, repeat), func(t *testing.T) {
					var calls, closed int32
					var methods, bodies []string
					transport := resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
						n := atomic.AddInt32(&calls, 1)
						body := ""
						if q.Body != nil {
							b, _ := io.ReadAll(q.Body)
							_ = q.Body.Close()
							body = strings.TrimSpace(string(b))
						}
						methods = append(methods, q.Method)
						bodies = append(bodies, body)
						if n == 1 && q.URL.RequestURI() != aclcreatePreparsePath(action) {
							t.Error("initial route changed")
						}
						if n == 1 {
							return &http.Response{StatusCode: code, Header: http.Header{"Location": []string{"/committed?SECRET=location"}}, Body: &resourceRedirectBody{Reader: strings.NewReader("BODY-SECRET"), closed: &closed}, Request: q}, nil
						}
						return &http.Response{StatusCode: 202, Header: make(http.Header), Body: &resourceRedirectBody{Reader: strings.NewReader(`{"uuid":"invented","objects":[{"uuid":"invented"}]}`), closed: &closed}, Request: q}, nil
					})
					c := sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic-secret"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: transport}))
					var value bool
					var r *sdk.Response
					var e error
					wantBody := aclcreatePreparseBody(action)
					value, r, e = aclcreatePreparseCall(c, action)
					t.Logf("actual calls=%d methods=%v bodies=%q closed=%d value=%v responsePresent=%v error=%v", calls, methods, bodies, closed, value, r != nil, e)
					if len(bodies) == 0 || bodies[0] != wantBody {
						t.Error("initial optional payload changed")
					}
					if calls != 1 || closed != 1 || value || r != nil || e != aclcreatePreparseSentinel(action) {
						t.Error("redirect admitted or unknown outcome not refused")
					}
				})
			}
		}
	}
}
