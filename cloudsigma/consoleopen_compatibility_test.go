package cloudsigma_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestConsoleOpenPreparseTypedAndRawErrors(t *testing.T) {
	for _, action := range []string{"open_console", "open_vnc"} {
		for _, payload := range []string{`[{"error_message":"denied","error_point":"server","error_type":"synthetic"}]`, `{"unknown":"raw-error"}`} {
			t.Run(action+"/"+payload, func(t *testing.T) {
				calls := 0
				c := sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
					calls++
					if q.Body != nil {
						_ = q.Body.Close()
					}
					return &http.Response{StatusCode: 403, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(payload)), Request: q}, nil
				})}))
				v, r, e := consoleOpenPreparseCall(c, action)
				if v || r == nil || r.StatusCode != 403 || e == nil || calls != 1 {
					t.Fatalf("error ownership: %v %v %v calls=%d", v, r, e, calls)
				}
				var typed *sdk.ErrorResponse
				if strings.HasPrefix(payload, "[") {
					if !errors.As(e, &typed) || typed.Response != r || !reflect.DeepEqual(typed.Errors, []sdk.Error{{Message: "denied", Point: "server", Type: "synthetic"}}) {
						t.Fatal("typed error changed", e)
					}
				} else {
					var raw *json.UnmarshalTypeError
					if !errors.As(e, &raw) || errors.As(e, &typed) {
						t.Fatal("legacy raw decoder error changed", e)
					}
				}
			})
		}
	}
}
