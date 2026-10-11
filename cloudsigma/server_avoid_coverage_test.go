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

type avoidCoverageWire struct {
	t                                        *testing.T
	action, encoded, decoded, body, response string
	code, calls                              int
	err                                      error
}

func (f *avoidCoverageWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	suffix := ""
	if f.encoded != "" {
		suffix = "&avoid=" + f.encoded
	}
	if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/servers/valid-id/action/?do="+f.action+suffix || q.URL.Fragment != "" {
		f.t.Fatal("URI")
	}
	want := map[string][]string{"do": {f.action}}
	if f.encoded != "" {
		want["avoid"] = []string{f.decoded}
	}
	if !reflect.DeepEqual(map[string][]string(q.URL.Query()), want) {
		f.t.Fatal("query multiplicity/order")
	}
	var raw []byte
	var e error
	if q.Body != nil {
		raw, e = io.ReadAll(q.Body)
	}
	if e != nil || strings.TrimSpace(string(raw)) != f.body {
		f.t.Fatal("wire body")
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.response)), Request: q}, nil
}
func avoidCoverageClient(f *avoidCoverageWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func callAvoid(c *Client, action string, ctx context.Context, opts ServerCloneRequest, ids []string) (interface{}, *Response, error) {
	if action == "start" {
		v, r, e := c.Servers.StartWithAvoid(ctx, "valid-id", ids)
		if v == nil {
			return nil, r, e
		}
		return v, r, e
	}
	v, r, e := c.Servers.CloneWithAvoid(ctx, "valid-id", opts, ids)
	if v == nil {
		return nil, r, e
	}
	return v, r, e
}
func TestAvoidRequestMatrix(t *testing.T) {
	for _, action := range []string{"start", "clone"} {
		t.Run(action, func(t *testing.T) {
			for _, tc := range []struct {
				name             string
				ids              []string
				encoded, decoded string
			}{{"nil", nil, "", ""}, {"empty", []string{}, "", ""}, {"single", []string{"server-a"}, "server-a", "server-a"}, {"ordered", []string{"server-b", "server-a"}, "server-b%2Cserver-a", "server-b,server-a"}, {"duplicates", []string{"server-b", "server-a", "server-b"}, "server-b%2Cserver-a%2Cserver-b", "server-b,server-a,server-b"}} {
				t.Run(tc.name, func(t *testing.T) {
					before := append([]string{}, tc.ids...)
					f := &avoidCoverageWire{t: t, action: action, encoded: tc.encoded, decoded: tc.decoded, body: `{}`, response: `{}`, code: 202}
					c := avoidCoverageClient(f)
					v, r, e := callAvoid(c, action, context.Background(), ServerCloneRequest{}, tc.ids)
					if e != nil || v == nil || r == nil || r.StatusCode != 202 || f.calls != 1 || !reflect.DeepEqual(append([]string{}, tc.ids...), before) {
						t.Fatal("request/preservation")
					}
					if len(tc.ids) == 0 {
						if action == "start" {
							_, r, e = c.Servers.Start(context.Background(), "valid-id")
						} else {
							_, r, e = c.Servers.Clone(context.Background(), "valid-id", ServerCloneRequest{})
						}
						if e != nil || r == nil || r.StatusCode != 202 || f.calls != 2 {
							t.Fatal("legacy percall")
						}
					}
				})
			}
		})
	}
}
func TestCloneAvoidPayloadMatrix(t *testing.T) {
	ids := []string{"server-a"}
	for _, tc := range []struct {
		name string
		opts ServerCloneRequest
		wire string
	}{{"nil", nil, ""}, {"empty", ServerCloneRequest{}, `{}`}, {"name-only", ServerCloneRequest{"name": json.RawMessage(`"name"`)}, `{"name":"name"}`}, {"false-only", ServerCloneRequest{"random_vnc_password": json.RawMessage(`false`)}, `{"random_vnc_password":false}`}} {
		t.Run(tc.name, func(t *testing.T) {
			before := make(ServerCloneRequest, len(tc.opts))
			for k, v := range tc.opts {
				before[k] = append(json.RawMessage(nil), v...)
			}
			f := &avoidCoverageWire{t: t, action: "clone", encoded: "server-a", decoded: "server-a", body: tc.wire, response: `{}`, code: 202}
			v, r, e := avoidCoverageClient(f).Servers.CloneWithAvoid(context.Background(), "valid-id", tc.opts, ids)
			if e != nil || v == nil || r == nil || r.StatusCode != 202 || f.calls != 1 || len(tc.opts) != len(before) || ids[0] != "server-a" {
				t.Fatal("payload")
			}
			for k, v := range before {
				if !reflect.DeepEqual(v, tc.opts[k]) {
					t.Fatal("byte mutation")
				}
			}
		})
	}
	for _, raw := range []json.RawMessage{nil, {}, json.RawMessage(`{`)} {
		opts := ServerCloneRequest{"bad": raw}
		f := &avoidCoverageWire{t: t}
		v, r, e := avoidCoverageClient(f).Servers.CloneWithAvoid(context.Background(), "valid-id", opts, ids)
		if v != nil || r != nil || e == nil || f.calls != 0 || !reflect.DeepEqual(opts["bad"], raw) || ids[0] != "server-a" {
			t.Fatal("invalid raw")
		}
	}
}
func TestAvoidResponseBoundaries(t *testing.T) {
	for _, action := range []string{"start", "clone"} {
		t.Run(action, func(t *testing.T) {
			f := &avoidCoverageWire{t: t, action: action, encoded: "server-a", decoded: "server-a", body: `{}`, code: 202}
			c := avoidCoverageClient(f)
			call := func() (interface{}, *Response, error) {
				return callAvoid(c, action, context.Background(), ServerCloneRequest{}, []string{"server-a"})
			}
			for _, code := range []int{403, 404, 500} {
				f.code = code
				f.response = `[{"error_message":"denied","error_point":"server","error_type":"synthetic"}]`
				n := f.calls
				v, r, e := call()
				var api *ErrorResponse
				if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || !reflect.DeepEqual(api.Errors, []Error{{Message: "denied", Point: "server", Type: "synthetic"}}) || f.calls != n+1 {
					t.Fatal("HTTP fidelity")
				}
			}
			f.code = 202
			bodies := []string{"", `{`, `[]`, `1`}
			if action == "clone" {
				bodies = append(bodies, `{"cpu":"bad"}`)
			}
			for _, body := range bodies {
				f.response = body
				n := f.calls
				v, r, e := call()
				if v != nil || r != nil || e == nil || f.calls != n+1 {
					t.Fatal("decode refusal")
				}
			}
			bodies = []string{`null`, `{}`}
			if action == "clone" {
				bodies = append(bodies, `{"objects":[]}`)
			}
			for _, body := range bodies {
				f.response = body
				n := f.calls
				v, r, e := call()
				var want interface{} = &ServerAction{}
				if action == "clone" {
					want = &Server{}
				}
				if e != nil || r == nil || r.StatusCode != 202 || !reflect.DeepEqual(v, want) || f.calls != n+1 {
					t.Fatal("permissive response")
				}
			}
			sentinel := errors.New("synthetic transport")
			f.err = sentinel
			n := f.calls
			v, r, e := call()
			if v != nil || r != nil || !errors.Is(e, sentinel) || f.calls != n+1 {
				t.Fatal("transport")
			}
			f.err = nil
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			n = f.calls
			v, r, e = callAvoid(c, action, ctx, ServerCloneRequest{}, []string{"server-a"})
			if v != nil || r != nil || !errors.Is(e, context.Canceled) || f.calls != n+1 {
				t.Fatal("cancellation")
			}
		})
	}
}
func TestCloneAvoidNestedLicenseCompatibility(t *testing.T) {
	// Synthetic SDK decoder compatibility fixture, NOT public server guarantees.
	f := &avoidCoverageWire{t: t, action: "clone", encoded: "server-a", decoded: "server-a", body: "", code: 202, response: `{"drives":[{"drive":{"licenses":[{"license":{"free":9007199254740993,"initial_purchase_requirement":0,"subsequent_purchase_requirement":2.5}}]}}]}`}
	v, r, e := avoidCoverageClient(f).Servers.CloneWithAvoid(context.Background(), "valid-id", nil, []string{"server-a"})
	if e != nil || r == nil || r.StatusCode != 202 || f.calls != 1 || v == nil || len(v.Drives) != 1 || v.Drives[0].Drive == nil || len(v.Drives[0].Drive.Licenses) != 1 || v.Drives[0].Drive.Licenses[0].License == nil {
		t.Fatal("nested decode")
	}
	l := v.Drives[0].Drive.Licenses[0].License
	if l.Free != "9007199254740993" || l.InitialPurchaseRequirement != "0" || l.SubsequentPurchaseRequirement != "2.5" {
		t.Fatal("numeric precision")
	}
}
