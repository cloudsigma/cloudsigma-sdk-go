package cloudsigma

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type powerIDWire struct {
	t          *testing.T
	calls      int
	id, action string
}

func (f *powerIDWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	f.t.Logf("observed method=%s path=%q query=%q", q.Method, q.URL.Path, q.URL.RawQuery)
	if f.id != "" {
		if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/servers/"+f.id+"/action/?do="+f.action || q.URL.Fragment != "" {
			f.t.Fatal("route", q.URL)
		}
		b, e := io.ReadAll(q.Body)
		if e != nil || strings.TrimSpace(string(b)) != "{}" {
			f.t.Fatal("body", e)
		}
	}
	return &http.Response{StatusCode: 202, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"action":"` + f.action + `","result":"success","uuid":"returned-id"}`)), Request: q}, nil
}
func TestServerPowerIdentifierPolicy(t *testing.T) {
	for _, action := range []string{"start", "stop", "shutdown"} {
		t.Run(action, func(t *testing.T) {
			f := &powerIDWire{t: t, action: action}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			call := c.Servers.Start
			if action == "stop" {
				call = c.Servers.Stop
			}
			if action == "shutdown" {
				call = c.Servers.Shutdown
			}
			for _, id := range []string{"", "..", "../../other/path", "id?do=stop&x=", "id#fragment", "a/b", "a\\b", "%2f", "%252f", "%2e%2e", "%252e%252e", "%00", "%250a", "x\n", "x\r", "x\t", "x\x00", " x", "x ", "é", "／"} {
				n := f.calls
				v, r, e := call(context.Background(), id)
				if v != nil || r != nil || e == nil || f.calls != n {
					t.Errorf("unsafe id %q: result=%v response=%v error=%v dispatchDelta=%d", id, v, r, e, f.calls-n)
				}
				if id == "" && !errors.Is(e, ErrEmptyArgument) {
					t.Error("empty diagnostic changed")
				}
			}
			for _, id := range []string{"d0ae4a09-13d5-4af2-aa12-56935035adef", "d79eeec6-9cc5-4131-9f04-36961c267a74", "Legacy_TOKEN-123"} {
				f.id = id
				n := f.calls
				v, r, e := call(context.Background(), id)
				if e != nil || v == nil || v.Action != action || v.Result != "success" || v.UUID != "returned-id" || r == nil || r.StatusCode != 202 || f.calls != n+1 {
					t.Fatal("valid ACK", v, r, e)
				}
			}
		})
	}
}
