package cloudsigma_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// Literal public examples establish ACK decoding, not completed console closure.
func TestConsoleClosePublicContract(t *testing.T) {
	for _, tc := range []struct{ action, id, ack, path string }{
		{"close_console", "e27eacc3-0f20-4c1f-9f7d-eac705bf2d02", "7a3a67e0-93e4-46db-a4a2-c0f886987dc0", "/api/2.0/servers/e27eacc3-0f20-4c1f-9f7d-eac705bf2d02/action/?do=close_console"},
		{"close_vnc", "42cb8991-a39a-4ed3-b549-efd6a1dedd48", "80751bc7-37cc-48c1-bb08-fd04b46090bf", "/api/2.0/servers/42cb8991-a39a-4ed3-b549-efd6a1dedd48/action/?do=close_vnc"},
	} {
		t.Run(tc.action, func(t *testing.T) {
			calls := 0
			hc := &http.Client{Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
				calls++
				b, e := io.ReadAll(q.Body)
				_ = q.Body.Close()
				if e != nil || q.Method != "POST" || q.URL.RequestURI() != tc.path || strings.TrimSpace(string(b)) != "{}" || q.Header.Get("Authorization") != "Bearer synthetic-contract" {
					t.Fatalf("public request mismatch %s %s %q", q.Method, q.URL, b)
				}
				return &http.Response{StatusCode: 202, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"action":"` + tc.action + `","result":"success","uuid":"` + tc.ack + `"}`)), Request: q}, nil
			})}
			c := sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic-contract"), sdk.WithEndpoint("https://example.invalid/api/2.0/"), sdk.WithHTTPClient(hc))
			var v *sdk.ServerAction
			var r *sdk.Response
			var e error
			if tc.action == "close_console" {
				v, r, e = c.Servers.CloseConsole(context.Background(), tc.id)
			} else {
				v, r, e = c.Servers.CloseVNC(context.Background(), tc.id)
			}
			if e != nil || r == nil || r.StatusCode != 202 || v == nil || *v != (sdk.ServerAction{Action: tc.action, Result: "success", UUID: tc.ack}) || calls != 1 {
				t.Fatalf("ACK mismatch value=%+v response=%v error=%v calls=%d", v, r, e, calls)
			}
		})
	}
}
