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

type jobsWire struct {
	t                            *testing.T
	method, path, body, response string
	code, calls                  int
}

func (f *jobsWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != f.method || q.URL.RequestURI() != f.path {
		f.t.Fatal("route")
	}
	var b []byte
	var e error
	if q.Body != nil {
		b, e = io.ReadAll(q.Body)
	}
	if e != nil || strings.TrimSpace(string(b)) != f.body {
		f.t.Fatal("request bytes")
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.response)), Request: q}, nil
}
func TestServerJobsActualMethods(t *testing.T) {
	raw, e := os.ReadFile("server_clone_response.json")
	if e != nil {
		t.Fatal(e)
	}
	var public map[string]json.RawMessage
	if e = json.Unmarshal(raw, &public); e != nil {
		t.Fatal(e)
	}
	for _, method := range []string{"Clone", "CloneWithAvoid", "Get", "List"} {
		t.Run(method, func(t *testing.T) {
			f := &jobsWire{t: t, method: "POST", path: "/api/2.0/servers/id/action/?do=clone", response: string(raw), code: 202}
			if method == "CloneWithAvoid" {
				f.path += "&avoid=other"
			}
			if method == "Get" {
				f.method = "GET"
				f.path = "/api/2.0/servers/id/"
				f.code = 200
			}
			if method == "List" {
				f.method = "GET"
				f.path = "/api/2.0/servers/detail/"
				f.response = `{"objects":[` + string(raw) + `]}`
				f.code = 200
			}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			var v *Server
			var r *Response
			switch method {
			case "Clone":
				v, r, e = c.Servers.Clone(context.Background(), "id", nil)
			case "CloneWithAvoid":
				v, r, e = c.Servers.CloneWithAvoid(context.Background(), "id", nil, []string{"other"})
			case "Get":
				v, r, e = c.Servers.Get(context.Background(), "id")
			case "List":
				var values []Server
				values, r, e = c.Servers.List(context.Background())
				if len(values) == 1 {
					v = &values[0]
				}
			}
			if e != nil || v == nil || r == nil || r.StatusCode != f.code || f.calls != 1 {
				t.Fatal("response")
			}
			var got, want interface{}
			if json.Unmarshal(v.Jobs, &got) != nil || json.Unmarshal(public["jobs"], &want) != nil || !reflect.DeepEqual(got, want) {
				t.Fatal("public jobs lost")
			}
		})
	}
}
func TestServerJobsRequestExclusion(t *testing.T) {
	s := Server{Name: "name", Jobs: json.RawMessage(`[ {"opaque":1} ]`)}
	plain := Server{Name: "name"}
	for _, pair := range [][2]interface{}{{s, plain}, {ServerCreateRequest{Servers: []Server{s}}, ServerCreateRequest{Servers: []Server{plain}}}, {ServerUpdateRequest{Server: &s}, ServerUpdateRequest{Server: &plain}}} {
		a, e := json.Marshal(pair[0])
		if e != nil {
			t.Fatal(e)
		}
		b, e := json.Marshal(pair[1])
		if e != nil || string(a) != string(b) {
			t.Fatal("jobs leaked into request")
		}
	}
	for _, method := range []string{"Create", "Update"} {
		f := &jobsWire{t: t, method: "POST", path: "/api/2.0/servers/", body: `{"objects":[{"name":"name"}]}`, response: `{"objects":[]}`, code: 201}
		if method == "Update" {
			f.method = "PUT"
			f.path = "/api/2.0/servers/id/"
			f.body = `{"name":"name"}`
			f.response = `{}`
			f.code = 200
		}
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
		var e error
		if method == "Create" {
			_, _, e = c.Servers.Create(context.Background(), &ServerCreateRequest{Servers: []Server{s}})
		} else {
			_, _, e = c.Servers.Update(context.Background(), "id", &ServerUpdateRequest{Server: &s})
		}
		if e != nil || f.calls != 1 {
			t.Fatal("request")
		}
	}
}
func TestServerJobsBoundariesAndLicense(t *testing.T) {
	f := &jobsWire{t: t, method: "GET", path: "/api/2.0/servers/id/", code: 200}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	for _, tc := range []struct{ body, want string }{{`{}`, ""}, {`null`, ""}, {`{"jobs":null}`, `null`}, {`{"jobs":[]}`, `[]`}, {`{"jobs":{"future":9007199254740993}}`, `{"future":9007199254740993}`}} {
		f.response = tc.body
		n := f.calls
		v, r, e := c.Servers.Get(context.Background(), "id")
		if e != nil || r == nil || v == nil || string(v.Jobs) != tc.want || f.calls != n+1 {
			t.Fatal("raw presence")
		}
	}
	// Synthetic nested-license decoder compatibility, not public jobs schema.
	f.response = `{"jobs":[{"opaque":false}],"drives":[{"drive":{"licenses":[{"license":{"free":9007199254740993}}]}}]}`
	v, _, e := c.Servers.Get(context.Background(), "id")
	if e != nil || v == nil || len(v.Drives) != 1 || v.Drives[0].Drive == nil || len(v.Drives[0].Drive.Licenses) != 1 || v.Drives[0].Drive.Licenses[0].License == nil || v.Drives[0].Drive.Licenses[0].License.Free != "9007199254740993" || string(v.Jobs) != `[{"opaque":false}]` {
		t.Fatal("license compatibility")
	}
	for _, bad := range []string{"", `{"jobs":`, `[]`} {
		f.response = bad
		v, r, e := c.Servers.Get(context.Background(), "id")
		if v != nil || r != nil || e == nil {
			t.Fatal("decode error")
		}
	}
	f.code = 403
	f.response = `[]`
	v, r, e := c.Servers.Get(context.Background(), "id")
	var api *ErrorResponse
	if v != nil || r == nil || r.StatusCode != 403 || !errors.As(e, &api) {
		t.Fatal("HTTP error")
	}
}
