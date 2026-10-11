package cloudsigma

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"
)

func TestServerJobsCreateUpdateResponses(t *testing.T) {
	for _, method := range []string{"Create", "Update"} {
		t.Run(method, func(t *testing.T) {
			f := &jobsWire{t: t, method: "POST", path: "/api/2.0/servers/", body: `{"objects":[{"name":"name"}]}`, code: 201, response: `{"objects":[{"jobs":[{"uuid":"job-a","opaque":9007199254740993}]},{"jobs":null},{"jobs":[]}]}`}
			if method == "Update" {
				f.method = "PUT"
				f.path = "/api/2.0/servers/id/"
				f.body = `{"name":"name"}`
				f.code = 200
				f.response = `{"jobs":[{"uuid":"job-a","opaque":9007199254740993}]}`
			}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			s := Server{Name: "name", Jobs: json.RawMessage(`{`)}
			call := func() ([]Server, *Response, error) {
				if method == "Create" {
					return c.Servers.Create(context.Background(), &ServerCreateRequest{Servers: []Server{s}})
				}
				v, r, e := c.Servers.Update(context.Background(), "id", &ServerUpdateRequest{Server: &s})
				if v == nil {
					return nil, r, e
				}
				return []Server{*v}, r, e
			}
			values, r, e := call()
			if e != nil || r == nil || r.StatusCode != f.code || f.calls != 1 || len(values) == 0 || string(values[0].Jobs) != `[{"uuid":"job-a","opaque":9007199254740993}]` {
				t.Fatal("jobs response")
			}
			if method == "Create" && (len(values) != 3 || string(values[1].Jobs) != "null" || string(values[2].Jobs) != "[]") {
				t.Fatal("distinct jobs")
			}
			for _, body := range []string{"", `{`, `[]`} {
				f.response = body
				n := f.calls
				v, r, e := call()
				if v != nil || r != nil || e == nil || f.calls != n+1 {
					t.Fatal("malformed response convention")
				}
			}
			f.code = 403
			f.response = `[{"error_message":"denied","error_point":"server","error_type":"synthetic"}]`
			n := f.calls
			v, r, e := call()
			var api *ErrorResponse
			if v != nil || r == nil || r.StatusCode != 403 || !errors.As(e, &api) || api.Response != r || !reflect.DeepEqual(api.Errors, []Error{{Message: "denied", Point: "server", Type: "synthetic"}}) || f.calls != n+1 {
				t.Fatal("error fidelity")
			}
		})
	}
}
func TestServerJobsMalformedExclusionAndPublicDecode(t *testing.T) {
	s := Server{Name: "name", Jobs: json.RawMessage(`{`)}
	plain := Server{Name: "name"}
	for _, pair := range [][2]interface{}{{s, plain}, {ServerCreateRequest{Servers: []Server{s}}, ServerCreateRequest{Servers: []Server{plain}}}, {ServerUpdateRequest{Server: &s}, ServerUpdateRequest{Server: &plain}}} {
		a, e := json.Marshal(pair[0])
		if e != nil {
			t.Fatal("malformed jobs must be excluded")
		}
		b, e := json.Marshal(pair[1])
		if e != nil || string(a) != string(b) {
			t.Fatal("request changed")
		}
	}
	var fresh Server
	if e := json.Unmarshal([]byte(`{"jobs":[1]}`), &fresh); e != nil || fresh.Jobs != nil {
		t.Fatal("public fresh decode")
	}
	before := string(s.Jobs)
	if e := json.Unmarshal([]byte(`{"jobs":null,"name":"new"}`), &s); e != nil || string(s.Jobs) != before || s.Name != "new" {
		t.Fatal("public prepopulated decode")
	}
}
func TestServerJobsOwnershipAndReuse(t *testing.T) {
	input := []byte(`{"jobs":[{"x":1}]}`)
	var a, b serverResponse
	if e := json.Unmarshal(input, &a); e != nil {
		t.Fatal(e)
	}
	if e := json.Unmarshal(input, &b); e != nil {
		t.Fatal(e)
	}
	for i := range input {
		input[i] = ' '
	}
	if string(a.Jobs) != `[{"x":1}]` || string(b.Jobs) != `[{"x":1}]` {
		t.Fatal("input alias")
	}
	a.Jobs[0] = ' '
	if string(b.Jobs) != `[{"x":1}]` {
		t.Fatal("independent results alias")
	}
	value := Server(b)
	copied := value
	copied.Jobs[0] = ' '
	if value.Jobs[0] != ' ' {
		t.Fatal("ordinary Server copies should share raw backing slice")
	}
	for _, tc := range []struct{ body, want string }{{`{}`, ""}, {`{"jobs":null}`, "null"}, {`{"jobs":[]}`, "[]"}, {`null`, ""}} {
		r := serverResponse(Server{Jobs: json.RawMessage(`[1]`), Name: "retained"})
		if e := json.Unmarshal([]byte(tc.body), &r); e != nil || string(r.Jobs) != tc.want || r.Name != "retained" {
			t.Fatal("deliberate reused decoder clearing")
		}
	}
}
