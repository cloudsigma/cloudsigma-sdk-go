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

type vlanRawPageWire struct {
	t           *testing.T
	path, body  string
	code, calls int
	err         error
}

func (f *vlanRawPageWire) RoundTrip(q *http.Request) (*http.Response, error) {
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
func TestVLANRawPages(t *testing.T) {
	for _, detail := range []bool{false, true} {
		name := "summary"
		path := "/api/2.0/vlans/"
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
					f := &vlanRawPageWire{t: t, path: path + tc.query, code: 200, body: `{"meta":{"limit":2,"offset":3,"total_count":9,"next":"https://evil.invalid/"},"objects":[{"uuid":"one","meta":{"name":"vlan"},"servers":[{"uuid":"server"}],"subscription":{"id":7}},{"uuid":"two"}]}`}
					c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
					var before ListOptions
					if tc.opts != nil {
						before = *tc.opts
					}
					var v []map[string]json.RawMessage
					var r *Response
					var e error
					if detail {
						v, r, e = c.VLANs.ListDetailRaw(context.Background(), tc.opts)
					} else {
						v, r, e = c.VLANs.ListSummaryRaw(context.Background(), tc.opts)
					}
					if e != nil || r == nil || r.StatusCode != 200 || r.Meta == nil || *r.Meta != (Meta{Limit: 2, Offset: 3, TotalCount: 9}) || len(v) != 2 || string(v[0]["uuid"]) != `"one"` || string(v[1]["uuid"]) != `"two"` || !definitionJSONEqual(string(v[0]["meta"]), `{"name":"vlan"}`) || !definitionJSONEqual(string(v[0]["servers"]), `[{"uuid":"server"}]`) || !definitionJSONEqual(string(v[0]["subscription"]), `{"id":7}`) || f.calls != 1 {
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
func TestVLANRawPageBoundaries(t *testing.T) {
	for _, detail := range []bool{false, true} {
		name := "summary"
		path := "/api/2.0/vlans/"
		if detail {
			name = "detail"
			path += "detail/"
		}
		t.Run(name, func(t *testing.T) {
			f := &vlanRawPageWire{t: t, path: path, code: 200}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			call := c.VLANs.ListSummaryRaw
			if detail {
				call = c.VLANs.ListDetailRaw
			}
			for _, opts := range []*ListOptions{{Limit: -1}, {Offset: -1}} {
				v, r, e := call(context.Background(), opts)
				if v != nil || r != nil || e == nil || f.calls != 0 {
					t.Fatal("negative")
				}
			}
			for _, body := range []string{`{"objects":[]}`, `{"objects":null}`, `{}`, `null`, ``, `{`, `[]`, `1`} {
				f.calls = 0
				f.body = body
				v, r, e := call(context.Background(), nil)
				bad := body == `` || body == `{` || body == `[]` || body == `1`
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
					if body == `[]` || body == `1` {
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

func TestVLANRawPagePublicAndOwnership(t *testing.T) {
	for _, detail := range []bool{false, true} {
		name := "summary"
		path := "/api/2.0/vlans/"
		if detail {
			name = "detail"
			path += "detail/"
		}
		t.Run(name, func(t *testing.T) {
			data, e := os.ReadFile("vlan_raw_" + name + "_page.json")
			if e != nil {
				t.Fatal(e)
			}
			var expected struct {
				Meta    Meta                         `json:"meta"`
				Objects []map[string]json.RawMessage `json:"objects"`
			}
			if json.Unmarshal(data, &expected) != nil {
				t.Fatal("fixture")
			}
			f := &vlanRawPageWire{t: t, path: path, code: 200, body: string(data)}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			call := c.VLANs.ListSummaryRaw
			if detail {
				call = c.VLANs.ListDetailRaw
			}
			v, r, e := call(context.Background(), nil)
			if e != nil || r == nil || r.StatusCode != 200 || r.Meta == nil || *r.Meta != expected.Meta || !reflect.DeepEqual(v, expected.Objects) || f.calls != 1 {
				t.Fatal("public preservation")
			}
			if detail {
				for k, w := range map[string]string{"bottom_vlan_tag": "null", "grantees": "[]", "permissions": "[]"} {
					if string(v[0][k]) != w {
						t.Fatal("field")
					}
				}
			}
			// Synthetic values are representation probes, not valid public field schemas.
			f.calls = 0
			f.body = `{"meta":{"limit":2,"offset":0,"total_count":2,"next":"https://evil.invalid/"},"extra":9,"objects":[{"n":9007199254740993,"f":false,"z":0,"s":"","a":[],"o":{},"nil":null},{"n":9007199254740993}]}`
			v, r, e = call(context.Background(), nil)
			if e != nil || r == nil || len(v) != 2 || f.calls != 1 || string(v[0]["n"]) != "9007199254740993" || string(v[0]["nil"]) != "null" || string(v[0]["f"]) != "false" || string(v[0]["z"]) != "0" || string(v[0]["s"]) != `""` || string(v[0]["a"]) != "[]" || string(v[0]["o"]) != "{}" {
				t.Fatal("synthetic")
			}
			again, _, e := call(context.Background(), nil)
			before := f.body
			v[0]["n"][0] = '1'
			if string(v[0]["n"]) != "1007199254740993" {
				t.Fatal("first raw value mutation did not occur")
			}
			if e != nil || f.body != before || string(v[1]["n"]) != "9007199254740993" || string(again[0]["n"]) != "9007199254740993" || f.calls != 2 {
				t.Fatal("independence")
			}
			for _, body := range []string{`{"objects":[null]}`, `{"objects":{}}`, `{"objects":[1]}`, `{"meta":{"limit":"bad"}}`} {
				f.calls = 0
				f.body = body
				v, r, e = call(context.Background(), nil)
				if body == `{"objects":[null]}` {
					if e != nil || r == nil || len(v) != 1 || v[0] != nil {
						t.Fatal("null element")
					}
				} else if e == nil || v != nil || r != nil {
					t.Fatal("wrong envelope")
				}
				if f.calls != 1 {
					t.Fatal("count")
				}
			}
		})
	}
}
