package cloudsigma_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// This identical oracle is also run with the two baseline production files.
// The fixture commits the first POST and would return an invented identity on
// follow-up. Refusal cannot establish whether that initial commit succeeded.
func TestCreatePreparseReplayAndPayload(t *testing.T) {
	for _, action := range []string{"drive", "server"} {
		for _, code := range []int{301, 302, 303, 307, 308} {
			for _, options := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%d/options=%v", action, code, options), func(t *testing.T) {
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
						if n == 1 && q.URL.RequestURI() != createPreparsePath(action) {
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
					wantBody := createPreparseBody(action)
					if !options {
						value, r, e = createPreparseCall(c, action)
					} else if action == "drive" {
						v, rr, ee := c.Drives.Create(context.Background(), &sdk.DriveCreateRequest{Drives: []sdk.Drive{{Name: "create", Meta: map[string]interface{}{"key": "value"}}}})
						value, r, e = v != nil, rr, ee
						wantBody = `{"objects":[{"meta":{"key":"value"},"name":"create"}]}`
					} else {
						v, rr, ee := c.Servers.Create(context.Background(), &sdk.ServerCreateRequest{Servers: []sdk.Server{{Name: "create", Meta: map[string]interface{}{"key": "value"}}}})
						value, r, e = v != nil, rr, ee
						wantBody = `{"objects":[{"meta":{"key":"value"},"name":"create"}]}`
					}
					t.Logf("actual calls=%d methods=%v bodies=%q closed=%d value=%v responsePresent=%v error=%v", calls, methods, bodies, closed, value, r != nil, e)
					if len(bodies) == 0 || bodies[0] != wantBody {
						t.Error("initial optional payload changed")
					}
					if calls != 1 || closed != 1 || value || r != nil || e != createPreparseSentinel(action) {
						t.Error("redirect admitted or unknown outcome not refused")
					}
				})
			}
		}
	}
}
