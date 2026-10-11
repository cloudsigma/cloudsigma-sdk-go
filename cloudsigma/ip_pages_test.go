package cloudsigma

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type ipPageWire struct {
	t           *testing.T
	path, body  string
	code, calls int
	err         error
}

func (f *ipPageWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "GET" || q.URL.RequestURI() != f.path || q.URL.Fragment != "" || q.ContentLength != 0 {
		f.t.Fatal("wire")
	}
	if q.Body != nil {
		b, e := io.ReadAll(q.Body)
		if e != nil || len(b) != 0 {
			f.t.Fatal("body")
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
func TestIPPages(t *testing.T) {
	for _, detail := range []bool{false, true} {
		name := "summary"
		path := "/api/2.0/ips/"
		if detail {
			name = "detail"
			path += "detail/"
		}
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				name, query string
				opts        *ListOptions
			}{{"nil", "", nil}, {"zero", "?limit=0", &ListOptions{}}, {"page", "?limit=2&offset=3", &ListOptions{Limit: 2, Offset: 3}}} {
				t.Run(tc.name, func(t *testing.T) {
					f := &ipPageWire{t: t, path: path + tc.query, code: 200, body: `{"meta":{"limit":2,"offset":3,"total_count":9,"next":"https://evil.invalid/"},"objects":[{"uuid":"192.0.2.10","meta":{"name":"ip"},"gateway":"192.0.2.1","netmask":24,"nameservers":["192.0.2.53"],"server":{"uuid":"server"}},{"uuid":"192.0.2.11"}]}`}
					c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
					var before ListOptions
					if tc.opts != nil {
						before = *tc.opts
					}
					var v []IP
					var r *Response
					var e error
					if detail {
						v, r, e = c.IPs.ListDetail(context.Background(), tc.opts)
					} else {
						v, r, e = c.IPs.ListSummary(context.Background(), tc.opts)
					}
					if e != nil || r == nil || r.StatusCode != 200 || r.Meta == nil || *r.Meta != (Meta{Limit: 2, Offset: 3, TotalCount: 9}) || len(v) != 2 || v[0].UUID != "192.0.2.10" || v[1].UUID != "192.0.2.11" || v[0].Meta["name"] != "ip" || v[0].Server == nil || v[0].Server.UUID != "server" || v[0].Gateway != "192.0.2.1" || v[0].Netmask != 24 || len(v[0].Nameservers) != 1 || v[0].Nameservers[0] != "192.0.2.53" || f.calls != 1 {
						t.Fatal("projection/count/meta")
					}
					if tc.opts != nil && *tc.opts != before {
						t.Fatal("mutation")
					}
				})
			}
		})
	}
}
func TestIPPageBoundaries(t *testing.T) {
	for _, detail := range []bool{false, true} {
		name := "summary"
		path := "/api/2.0/ips/"
		if detail {
			name = "detail"
			path += "detail/"
		}
		t.Run(name, func(t *testing.T) {
			f := &ipPageWire{t: t, path: path, code: 200}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			call := c.IPs.ListSummary
			if detail {
				call = c.IPs.ListDetail
			}
			for _, opts := range []*ListOptions{{Limit: -1}, {Offset: -1}} {
				v, r, e := call(context.Background(), opts)
				if v != nil || r != nil || e == nil || f.calls != 0 {
					t.Fatal("negative")
				}
			}
			for _, body := range []string{`{"objects":[]}`, `{"objects":null}`, `{}`, `null`, ``, `{`, `[]`, `1`, `{"objects":{}}`, `{"objects":[{"netmask":"bad"}]}`, `{"meta":{"limit":"bad"}}`} {
				f.calls = 0
				f.body = body
				v, r, e := call(context.Background(), nil)
				bad := body == `` || body == `{` || body == `[]` || body == `1` || body == `{"objects":{}}` || body == `{"objects":[{"netmask":"bad"}]}` || body == `{"meta":{"limit":"bad"}}`
				if f.calls != 1 {
					t.Fatal("count")
				}
				if bad {
					if v != nil || r != nil || e == nil {
						t.Fatal("decode error")
					}
					if body == `` && !errors.Is(e, io.EOF) {
						t.Fatal("EOF")
					}
					if body == `{` && !errors.Is(e, io.ErrUnexpectedEOF) {
						t.Fatal("incomplete")
					}
					if body == `[]` || body == `1` || body == `{"objects":{}}` || body == `{"objects":[{"netmask":"bad"}]}` || body == `{"meta":{"limit":"bad"}}` {
						var typed *json.UnmarshalTypeError
						if !errors.As(e, &typed) {
							t.Fatal("type")
						}
					}
				} else {
					if e != nil || r == nil {
						t.Fatal("success")
					}
					if body == `{"objects":[]}` {
						if v == nil || len(v) != 0 {
							t.Fatal("empty")
						}
					} else if v != nil {
						t.Fatal("null/missing")
					}
				}
			}
			for _, code := range []int{403, 404, 500} {
				f.calls = 0
				f.code = code
				f.body = `[{"error_message":"denied","error_point":"limit","error_type":"forbidden"}]`
				v, r, e := call(context.Background(), nil)
				var api *ErrorResponse
				if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0] != (Error{Message: "denied", Point: "limit", Type: "forbidden"}) || f.calls != 1 {
					t.Fatal("API error")
				}
			}
			for _, cancel := range []bool{false, true} {
				f.calls = 0
				sentinel := errors.New("transport")
				f.err = sentinel
				ctx := context.Background()
				if cancel {
					var stop context.CancelFunc
					ctx, stop = context.WithCancel(ctx)
					stop()
					sentinel = context.Canceled
				}
				v, r, e := call(ctx, nil)
				if v != nil || r != nil || !errors.Is(e, sentinel) || f.calls != 1 {
					t.Fatal("transport/context")
				}
			}
		})
	}
}
