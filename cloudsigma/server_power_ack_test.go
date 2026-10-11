package cloudsigma

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type powerWire struct {
	t            *testing.T
	action, body string
	code, calls  int
	err          error
}

func (f *powerWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/servers/test-id/action/?do="+f.action || q.URL.Fragment != "" {
		f.t.Fatal("route", q.URL)
	}
	var raw []byte
	var e error
	if q.Body != nil {
		raw, e = io.ReadAll(q.Body)
	}
	if e != nil || strings.TrimSpace(string(raw)) != "{}" {
		f.t.Errorf("documented body expected {}, got %q", raw)
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.body)), Request: q}, nil
}
func TestServerPowerACK(t *testing.T) {
	for _, action := range []string{"start", "stop", "shutdown"} {
		t.Run(action, func(t *testing.T) {
			f := &powerWire{t: t, action: action, code: 202}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			call := c.Servers.Start
			if action == "stop" {
				call = c.Servers.Stop
			}
			if action == "shutdown" {
				call = c.Servers.Shutdown
			}
			want := ServerAction{Action: action, Result: "success", UUID: "returned-id"}
			raw, e := json.Marshal(want)
			if e != nil {
				t.Fatal(e)
			}
			f.body = string(raw)
			v, r, e := call(context.Background(), "test-id")
			if e != nil || r == nil || r.StatusCode != 202 || !reflect.DeepEqual(v, &want) || f.calls != 1 {
				t.Fatal("ACK", v, r, e)
			}
			if v, r, e = call(context.Background(), ""); v != nil || r != nil || !errors.Is(e, ErrEmptyArgument) || f.calls != 1 {
				t.Fatal("emptyID", v, r, e)
			}
			for _, code := range []int{403, 404, 500} {
				f.code = code
				f.body = `[{"error_message":"denied","error_point":"server","error_type":"synthetic"}]`
				n := f.calls
				v, r, e = call(context.Background(), "test-id")
				var api *ErrorResponse
				if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || !reflect.DeepEqual(api.Errors, []Error{{Message: "denied", Point: "server", Type: "synthetic"}}) || f.calls != n+1 {
					t.Fatal("HTTP", v, r, e)
				}
			}
			f.code = 202
			for _, body := range []string{"", `{`, `[]`, `1`} {
				f.body = body
				n := f.calls
				v, r, e = call(context.Background(), "test-id")
				if v != nil || r != nil || e == nil || f.calls != n+1 {
					t.Fatal("decode", v, r, e)
				}
			}
			for _, body := range []string{`null`, `{}`} {
				f.body = body
				n := f.calls
				v, r, e = call(context.Background(), "test-id")
				if e != nil || r == nil || !reflect.DeepEqual(v, &ServerAction{}) || f.calls != n+1 {
					t.Fatal("permissive", v, r, e)
				}
			}
			sentinel := errors.New("power transport")
			f.err = sentinel
			n := f.calls
			v, r, e = call(context.Background(), "test-id")
			if v != nil || r != nil || !errors.Is(e, sentinel) || f.calls != n+1 {
				t.Fatal("transport", v, r, e)
			}
			f.err = nil
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			n = f.calls
			v, r, e = call(ctx, "test-id")
			if v != nil || r != nil || !errors.Is(e, context.Canceled) || f.calls != n+1 {
				t.Fatal("cancel", v, r, e)
			}
		})
	}
}
