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

type kvmWire struct {
	t       *testing.T
	want    string
	code    int
	body    string
	calls   int
	failure error
}

func (f *kvmWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "PUT" || q.URL.RequestURI() != "/api/2.0/servers/server/" {
		f.t.Fatalf("route %s %s", q.Method, q.URL)
	}
	b, e := io.ReadAll(q.Body)
	if e != nil {
		f.t.Fatal(e)
	}
	var got, want map[string]json.RawMessage
	if json.Unmarshal(b, &got) != nil || json.Unmarshal([]byte(f.want), &want) != nil || !reflect.DeepEqual(got, want) {
		f.t.Fatalf("body %s want %s", b, f.want)
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	if f.failure != nil {
		return nil, f.failure
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.body)), Request: q}, nil
}
func TestKVMOptionsPresence(t *testing.T) {
	yes, no := true, false
	for _, tc := range []struct {
		name      string
		cpu, numa *bool
		want      string
	}{
		{"absent", nil, nil, `{}`}, {"cpu_false", &no, nil, `{"cpus_instead_of_cores":false}`}, {"cpu_true", &yes, nil, `{"cpus_instead_of_cores":true}`},
		{"numa_false", nil, &no, `{"enable_numa":false}`}, {"numa_true", nil, &yes, `{"enable_numa":true}`},
		{"both_false", &no, &no, `{"cpus_instead_of_cores":false,"enable_numa":false}`}, {"both_true", &yes, &yes, `{"cpus_instead_of_cores":true,"enable_numa":true}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &kvmWire{t: t, want: tc.want, code: 200, body: `{"uuid":"server","status":"stopped","cpus_instead_of_cores":false,"enable_numa":false,"jobs":[]}`}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			opts := &ServerKVMOptions{CPUsInsteadOfCores: tc.cpu, EnableNuma: tc.numa}
			before := *opts
			s, r, e := c.Servers.UpdateKVMOptions(context.Background(), "server", opts)
			if e != nil || r == nil || r.StatusCode != 200 || s == nil || s.UUID != "server" || s.Status != "stopped" || s.CPUsInsteadOfCores || s.EnableNuma || string(s.Jobs) != "[]" || f.calls != 1 {
				t.Fatalf("response %v %v %v", s, r, e)
			}
			if *opts != before || !yes || no {
				t.Fatal("input mutated")
			}
		})
	}
}
func TestKVMOptionsFailures(t *testing.T) {
	f := &kvmWire{t: t, want: `{}`, code: 403, body: `[{"error_message":"denied","error_point":"server","error_type":"permission"}]`}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	s, r, e := c.Servers.UpdateKVMOptions(context.Background(), "server", &ServerKVMOptions{})
	var api *ErrorResponse
	if s != nil || r == nil || r.StatusCode != 403 || !errors.As(e, &api) || api.Response != r || !reflect.DeepEqual(api.Errors, []Error{{Message: "denied", Point: "server", Type: "permission"}}) || f.calls != 1 {
		t.Fatal("HTTP error fidelity")
	}
	for _, body := range []string{"", `{`, `[]`, `{"enable_numa":"false"}`} {
		f.code = 200
		f.body = body
		f.calls = 0
		s, _, e = c.Servers.UpdateKVMOptions(context.Background(), "server", &ServerKVMOptions{})
		if e == nil || s != nil || f.calls != 1 {
			t.Fatalf("malformed %q", body)
		}
	}
	f.calls = 0
	for _, id := range []string{"", "../escape", "server?x=1"} {
		_, _, e = c.Servers.UpdateKVMOptions(context.Background(), id, &ServerKVMOptions{})
		if e == nil {
			t.Fatal("identifier")
		}
	}
	_, _, e = c.Servers.UpdateKVMOptions(context.Background(), "server", nil)
	if !errors.Is(e, ErrEmptyPayloadNotAllowed) || f.calls != 0 {
		t.Fatal("nil/zero dispatch")
	}
	sentinel := errors.New("synthetic transport")
	f.failure = sentinel
	_, _, e = c.Servers.UpdateKVMOptions(context.Background(), "server", &ServerKVMOptions{})
	if !errors.Is(e, sentinel) {
		t.Fatal("transport")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, e = c.Servers.UpdateKVMOptions(ctx, "server", &ServerKVMOptions{})
	if !errors.Is(e, context.Canceled) {
		t.Fatal("cancellation")
	}
	// 202 is not the documented edit success status. Characterize the common
	// transport's acceptance without claiming an applied runtime change.
	f.failure = nil
	f.code = 202
	f.body = `{"uuid":"server","status":"starting"}`
	s, r, e = c.Servers.UpdateKVMOptions(context.Background(), "server", &ServerKVMOptions{})
	if e != nil || r.StatusCode != 202 || s.Status != "starting" {
		t.Fatal("202 acknowledgment characterization")
	}
}
