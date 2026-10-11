package cloudsigma

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type avoidWire struct {
	t                              *testing.T
	action, suffix, body, response string
	calls, code                    int
	err                            error
	wire                           string
}

func (f *avoidWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	var raw []byte
	var e error
	if q.Body != nil {
		raw, e = io.ReadAll(q.Body)
	}
	if e != nil {
		f.t.Fatal(e)
	}
	f.wire = q.Method + " " + q.URL.RequestURI() + " " + strings.TrimSpace(string(raw))
	if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/servers/valid-id/action/?do="+f.action+f.suffix || q.URL.Fragment != "" || strings.TrimSpace(string(raw)) != f.body {
		f.t.Fatal("wire mismatch")
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.response)), Request: q}, nil
}
func TestServerAvoidMethods(t *testing.T) {
	for _, action := range []string{"start", "clone"} {
		t.Run(action, func(t *testing.T) {
			f := &avoidWire{t: t, action: action, body: `{}`, code: 202, response: `{"uuid":"result","action":"start","result":"success","name":"cloned"}`}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			opts := ServerCloneRequest{}
			call := func(ctx context.Context, id string, ids []string) (*Response, error) {
				if action == "start" {
					v, r, e := c.Servers.StartWithAvoid(ctx, id, ids)
					if e != nil && v != nil {
						t.Fatal("error result")
					}
					if e == nil && (v.UUID != "result" || v.Action != "start" || v.Result != "success") {
						t.Fatal("ack")
					}
					return r, e
				}
				v, r, e := c.Servers.CloneWithAvoid(ctx, id, opts, ids)
				if e != nil && v != nil {
					t.Fatal("error result")
				}
				if e == nil && (v.UUID != "result" || v.Name != "cloned") {
					t.Fatal("definition")
				}
				return r, e
			}
			for _, ids := range [][]string{nil, {}} {
				n := f.calls
				r, e := call(context.Background(), "valid-id", ids)
				if e != nil || r.StatusCode != 202 || f.calls != n+1 {
					t.Fatal("no options")
				}
				wire := f.wire
				if action == "start" {
					_, _, e = c.Servers.Start(context.Background(), "valid-id")
				} else {
					_, _, e = c.Servers.Clone(context.Background(), "valid-id", opts)
				}
				if e != nil || wire != f.wire {
					t.Fatal("legacy equivalence")
				}
			}
			ids := []string{"bb1d5184-ebcc-4f33-867e-db654eb2d17e", "dc3dd6d4-9b2d-44e6-bc40-e927950e8b77"}
			before := append([]string(nil), ids...)
			f.suffix = "&avoid=" + url.QueryEscape(strings.Join(ids, ","))
			n := f.calls
			r, e := call(context.Background(), "valid-id", ids)
			if e != nil || r.StatusCode != 202 || f.calls != n+1 || !reflect.DeepEqual(ids, before) {
				t.Fatal("ordered hints")
			}
			if action == "clone" {
				opts = ServerCloneRequest{"name": json.RawMessage(`""`), "random_vnc_password": json.RawMessage(`false`), "unknown": json.RawMessage(`9007199254740993`)}
				f.body = `{"name":"","random_vnc_password":false,"unknown":9007199254740993}`
				b, _ := json.Marshal(opts)
				_, e = call(context.Background(), "valid-id", ids)
				a, _ := json.Marshal(opts)
				if e != nil || string(a) != string(b) {
					t.Fatal("payload preservation")
				}
			}
			for _, bad := range []string{"", "..", "a/b", "a?do=stop", "a#x", "%2f", "%252f", "a\\b", "x\n", " x", "é", "a,b"} {
				n = f.calls
				r, e = call(context.Background(), "valid-id", []string{"safe", bad})
				if r != nil || e == nil || f.calls != n {
					t.Fatal("invalid hint dispatch")
				}
				r, e = call(context.Background(), bad, ids)
				if r != nil || e == nil || f.calls != n {
					t.Fatal("invalid target dispatch")
				}
			}
			for _, code := range []int{403, 500} {
				f.code = code
				f.response = `[]`
				n = f.calls
				r, e = call(context.Background(), "valid-id", ids)
				var api *ErrorResponse
				if r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || f.calls != n+1 {
					t.Fatal("HTTP")
				}
			}
			sentinel := errors.New("synthetic")
			f.err = sentinel
			n = f.calls
			r, e = call(context.Background(), "valid-id", ids)
			if r != nil || !errors.Is(e, sentinel) || f.calls != n+1 {
				t.Fatal("transport")
			}
			f.err = nil
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			n = f.calls
			r, e = call(ctx, "valid-id", ids)
			if r != nil || !errors.Is(e, context.Canceled) || f.calls != n+1 {
				t.Fatal("context")
			}
		})
	}
}
