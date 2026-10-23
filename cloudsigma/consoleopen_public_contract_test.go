package cloudsigma_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// Public examples establish acknowledgment decoding, not connection readiness.
// Returned capability URLs remain data: no connection or password is sent.
func TestConsoleOpenPublicContract(t *testing.T) {
	for _, tc := range []struct{ action, id, ack, path, key, endpoint string }{
		{"open_console", "e27eacc3-0f20-4c1f-9f7d-eac705bf2d02", "7a3a67e0-93e4-46db-a4a2-c0f886987dc0", "/api/2.0/servers/e27eacc3-0f20-4c1f-9f7d-eac705bf2d02/action/?do=open_console", "console_url", "csconsole://direct.wdc.alpha3cloud.com:43835"},
		{"open_vnc", "42cb8991-a39a-4ed3-b549-efd6a1dedd48", "80751bc7-37cc-48c1-bb08-fd04b46090bf", "/api/2.0/servers/42cb8991-a39a-4ed3-b549-efd6a1dedd48/action/?do=open_vnc", "vnc_url", "vnc://direct.zrh.cloudsigma.com:42714"},
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
				return &http.Response{StatusCode: 202, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"action":"` + tc.action + `","result":"success","uuid":"` + tc.ack + `","` + tc.key + `":"` + tc.endpoint + `"}`)), Request: q}, nil
			})}
			c := sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic-contract"), sdk.WithEndpoint("https://example.invalid/api/2.0/"), sdk.WithHTTPClient(hc))
			if tc.action == "open_console" {
				v, r, e := c.Servers.OpenConsole(context.Background(), tc.id)
				if e != nil || r == nil || r.StatusCode != 202 || v == nil || *v != (sdk.ServerConsoleAction{Action: tc.action, Result: "success", UUID: tc.ack, ConsoleURL: tc.endpoint}) || calls != 1 {
					t.Fatalf("ACK mismatch %+v %v %v calls=%d", v, r, e, calls)
				}
			} else {
				v, r, e := c.Servers.OpenVNC(context.Background(), tc.id)
				if e != nil || r == nil || r.StatusCode != 202 || v == nil || *v != (sdk.ServerVNCAction{Action: tc.action, Result: "success", UUID: tc.ack, VNCURL: tc.endpoint}) || calls != 1 {
					t.Fatalf("ACK mismatch %+v %v %v calls=%d", v, r, e, calls)
				}
			}
		})
	}
}
