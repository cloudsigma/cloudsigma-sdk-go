package cloudsigma

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type serverPageWire struct {
	t           *testing.T
	path, body  string
	code, calls int
	err         error
}

func (f *serverPageWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "GET" || q.URL.RequestURI() != f.path || q.URL.Fragment != "" {
		f.t.Fatal("route")
	}
	if q.Body != nil {
		b, e := io.ReadAll(q.Body)
		if e != nil || len(b) != 0 {
			f.t.Fatal("GET body")
		}
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.body)), Request: q}, nil
}
func TestServerListPages(t *testing.T) {
	for _, detail := range []bool{false, true} {
		name := "summary"
		path := "/api/2.0/servers/"
		if detail {
			name = "detail"
			path += "detail/"
		}
		t.Run(name, func(t *testing.T) {
			f := &serverPageWire{t: t, path: path, code: 200, body: `{"meta":{"limit":2,"offset":3,"total_count":9},"objects":[{"uuid":"id","jobs":[{"opaque":9007199254740993}],"drives":[{"drive":{"licenses":[{"license":{"free":9007199254740993}}]}}]}]}`}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			call := c.Servers.ListSummary
			if detail {
				call = c.Servers.ListDetail
			}
			for _, tc := range []struct {
				opts  *ServerListOptions
				query string
			}{{nil, ""}, {&ServerListOptions{}, "?limit=0"}, {&ServerListOptions{ListOptions: ListOptions{Limit: 2, Offset: 3}, Name: "a & b", NameContains: "x/y", Tag: "tag?x", UUID: "id#x"}, "?limit=2&name=a+%26+b&name__contains=x%2Fy&offset=3&tag=tag%3Fx&uuid=id%23x"}} {
				f.path = path + tc.query
				var before ServerListOptions
				if tc.opts != nil {
					before = *tc.opts
				}
				n := f.calls
				v, r, e := call(context.Background(), tc.opts)
				if e != nil || r == nil || r.StatusCode != 200 || f.calls != n+1 || len(v) != 1 || !reflect.DeepEqual(r.Meta, &Meta{Limit: 2, Offset: 3, TotalCount: 9}) || string(v[0].Jobs) != `[{"opaque":9007199254740993}]` || v[0].Drives[0].Drive.Licenses[0].License.Free != "9007199254740993" {
					t.Fatal("page projection")
				}
				if tc.opts != nil && *tc.opts != before {
					t.Fatal("options mutation")
				}
			}
			f.path = path
			n := f.calls
			for _, opts := range []*ServerListOptions{{ListOptions: ListOptions{Limit: -1}}, {ListOptions: ListOptions{Offset: -1}}} {
				v, r, e := call(context.Background(), opts)
				if v != nil || r != nil || e == nil || f.calls != n {
					t.Fatal("negative dispatch")
				}
			}
			for _, body := range []string{`{"objects":[]}`, `{}`, `null`, `{"meta":{"next":"https://untrusted.invalid/","previous":"/other/"},"objects":[]}`} {
				f.body = body
				n = f.calls
				v, r, e := call(context.Background(), nil)
				if e != nil || r == nil || len(v) != 0 || f.calls != n+1 {
					t.Fatal("empty/no traversal")
				}
			}
			for _, body := range []string{"", `{`, `[]`, `{"meta":{"limit":"bad"}}`} {
				f.body = body
				n = f.calls
				v, r, e := call(context.Background(), nil)
				if v != nil || r != nil || e == nil || f.calls != n+1 {
					t.Fatal("decode")
				}
			}
			for _, code := range []int{403, 404, 500} {
				f.code = code
				f.body = `[{"error_message":"denied","error_point":"server","error_type":"synthetic"}]`
				n = f.calls
				v, r, e := call(context.Background(), nil)
				var api *ErrorResponse
				if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || !reflect.DeepEqual(api.Errors, []Error{{Message: "denied", Point: "server", Type: "synthetic"}}) || f.calls != n+1 {
					t.Fatal("HTTP")
				}
			}
			sentinel := errors.New("transport")
			f.err = sentinel
			n = f.calls
			v, r, e := call(context.Background(), nil)
			if v != nil || r != nil || !errors.Is(e, sentinel) || f.calls != n+1 {
				t.Fatal("transport")
			}
			f.err = nil
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			n = f.calls
			v, r, e = call(ctx, nil)
			if v != nil || r != nil || !errors.Is(e, context.Canceled) || f.calls != n+1 {
				t.Fatal("cancel")
			}
		})
	}
}
