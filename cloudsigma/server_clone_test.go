package cloudsigma

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
)

type cloneWire struct {
	t           *testing.T
	want        ServerCloneRequest
	body        string
	code, calls int
	err         error
}

func (f *cloneWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/servers/ee11b9d1-e837-4f27-91a4-d1ae4c621bd1/action/?do=clone" || q.URL.Fragment != "" {
		f.t.Fatal("route")
	}
	var raw []byte
	var e error
	if q.Body != nil {
		raw, e = io.ReadAll(q.Body)
	}
	if e != nil {
		f.t.Fatal(e)
	}
	if f.want == nil {
		if len(raw) != 0 {
			f.t.Fatal("optional body")
		}
	} else {
		var got ServerCloneRequest
		if e = json.Unmarshal(raw, &got); e != nil {
			f.t.Fatal(e)
		}
		if !reflect.DeepEqual(got, f.want) {
			f.t.Fatal("payload mismatch")
		}
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.body)), Request: q}, nil
}

const cloneID = "ee11b9d1-e837-4f27-91a4-d1ae4c621bd1"

func cloneClient(f *cloneWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func TestServerCloneWireAndOptionality(t *testing.T) {
	raw, e := os.ReadFile("server_clone_response.json")
	if e != nil {
		t.Fatal(e)
	}
	var expected serverResponse
	if e = json.Unmarshal(raw, &expected); e != nil {
		t.Fatal(e)
	}
	for _, options := range []ServerCloneRequest{nil, {}, {"name": json.RawMessage(`"test_cloned_server_name"`), "random_vnc_password": json.RawMessage(`true`)}, {"name": json.RawMessage(`""`), "random_vnc_password": json.RawMessage(`false`), "unknown": json.RawMessage(`null`), "zero": json.RawMessage(`0`), "array": json.RawMessage(`[]`)}} {
		before, _ := json.Marshal(options)
		f := &cloneWire{t: t, want: options, body: string(raw), code: 202}
		v, r, e := cloneClient(f).Servers.Clone(context.Background(), cloneID, options)
		after, _ := json.Marshal(options)
		if e != nil || r == nil || r.StatusCode != 202 || !reflect.DeepEqual(v, (*Server)(&expected)) || f.calls != 1 || string(before) != string(after) {
			t.Fatal("clone response or mutation")
		}
		if v.UUID != "2d69b07c-c7a3-4e25-88d2-c7055217e1fb" || v.Name != "test_cloned_server_name" || v.CPU != 1000 {
			t.Fatal("definition fields")
		}
	}
}
func TestServerCloneInvalidZeroDispatch(t *testing.T) {
	f := &cloneWire{t: t}
	c := cloneClient(f)
	for _, id := range []string{"", "..", "a/b", "a?b", "a#b", "%2f", "%252f", "a\\b", "x\n", "é"} {
		v, r, e := c.Servers.Clone(context.Background(), id, nil)
		if v != nil || r != nil || e == nil || f.calls != 0 {
			t.Fatal("invalid ID")
		}
	}
	for _, bad := range []ServerCloneRequest{{"x": nil}, {"x": json.RawMessage{}}, {"x": json.RawMessage(`{`)}} {
		v, r, e := c.Servers.Clone(context.Background(), cloneID, bad)
		if v != nil || r != nil || e == nil || f.calls != 0 {
			t.Fatal("invalid JSON")
		}
	}
}
func TestServerCloneErrorsAndDecoder(t *testing.T) {
	f := &cloneWire{t: t}
	c := cloneClient(f)
	for _, code := range []int{403, 404, 500} {
		f.code = code
		f.body = `[{"error_message":"denied","error_point":"server","error_type":"synthetic"}]`
		n := f.calls
		v, r, e := c.Servers.Clone(context.Background(), cloneID, nil)
		var api *ErrorResponse
		if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || !reflect.DeepEqual(api.Errors, []Error{{Message: "denied", Point: "server", Type: "synthetic"}}) || f.calls != n+1 {
			t.Fatal("HTTP fidelity")
		}
	}
	f.code = 202
	for _, body := range []string{"", `{`, `[]`, `1`, `{"cpu":"bad"}`} {
		f.body = body
		n := f.calls
		v, r, e := c.Servers.Clone(context.Background(), cloneID, nil)
		if v != nil || r != nil || e == nil || f.calls != n+1 {
			t.Fatal("decode error")
		}
	}
	for _, body := range []string{`null`, `{}`, `{"objects":[]}`} {
		f.body = body
		v, r, e := c.Servers.Clone(context.Background(), cloneID, nil)
		if e != nil || r == nil || !reflect.DeepEqual(v, &Server{}) {
			t.Fatal("permissive decoder")
		}
	}
	sentinel := errors.New("synthetic transport")
	f.err = sentinel
	n := f.calls
	v, r, e := c.Servers.Clone(context.Background(), cloneID, nil)
	if v != nil || r != nil || !errors.Is(e, sentinel) || f.calls != n+1 {
		t.Fatal("transport")
	}
	f.err = nil
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n = f.calls
	v, r, e = c.Servers.Clone(ctx, cloneID, nil)
	if v != nil || r != nil || !errors.Is(e, context.Canceled) || f.calls != n+1 {
		t.Fatal("context")
	}
}
