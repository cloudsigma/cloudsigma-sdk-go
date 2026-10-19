package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"testing"
)

type routeRawWire struct {
	t           *testing.T
	data        []byte
	code, calls int
	err         error
}

func (f *routeRawWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/routes/id/" || q.URL.Fragment != "" || q.ContentLength != 0 {
		f.t.Fatal("route")
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
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(f.data)), Request: q}, nil
}
func TestRouterRouteRawGetPublic(t *testing.T) {
	data, e := os.ReadFile("router_route_raw_public.json")
	if e != nil {
		t.Fatal(e)
	}
	var expected map[string]json.RawMessage
	if json.Unmarshal(data, &expected) != nil {
		t.Fatal("fixture")
	}
	f := &routeRawWire{t: t, data: data, code: 200}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	v, r, e := c.VirtualRouterRoutes.GetRaw(context.Background(), "id")
	if e != nil || r == nil || r.StatusCode != 200 || !reflect.DeepEqual(v, expected) || f.calls != 1 {
		t.Fatal("public preservation")
	}
	if string(v["uuid"]) != `"6251235d-c5a6-4422-a9e5-f6b4308f4f4a"` {
		t.Fatal("public ID")
	}
	var jobs []map[string]json.RawMessage
	if json.Unmarshal(v["jobs"], &jobs) != nil || len(jobs) != 1 || string(jobs[0]["uuid"]) != `"9e9a97ec-2a84-4938-a390-0a08a40d3fb7"` || string(jobs[0]["resource_uri"]) != `"/api/2.0/jobs/9e9a97ec-2a84-4938-a390-0a08a40d3fb7/"` {
		t.Fatal("public nested job")
	}
	before := append([]byte(nil), data...)
	v["uuid"][1] = 'X'
	if string(v["uuid"]) != `"X251235d-c5a6-4422-a9e5-f6b4308f4f4a"` {
		t.Fatal("mutation")
	}
	if !bytes.Equal(data, before) {
		t.Fatal("result aliases input")
	}
	snapshot := append([]byte(nil), v["uuid"]...)
	data[0] = '!'
	if !bytes.Equal(v["uuid"], snapshot) {
		t.Fatal("input aliases result")
	}

}
func TestRouterRouteRawGetSynthetic(t *testing.T) {
	f := &routeRawWire{t: t, code: 200, data: []byte(`{"synthetic_number":0,"synthetic_nested":[{"unknown":9007199254740993}],"synthetic_empty":[],"false":false,"empty":"","null":null,"next":"https://evil.invalid/"}`)}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	v, r, e := c.VirtualRouterRoutes.GetRaw(context.Background(), "id")
	want := map[string]json.RawMessage{"synthetic_number": json.RawMessage(`0`), "synthetic_nested": json.RawMessage(`[{"unknown":9007199254740993}]`), "synthetic_empty": json.RawMessage(`[]`), "false": json.RawMessage(`false`), "empty": json.RawMessage(`""`), "null": json.RawMessage(`null`), "next": json.RawMessage(`"https://evil.invalid/"`)}
	if e != nil || r == nil || !reflect.DeepEqual(v, want) || f.calls != 1 {
		t.Fatal("synthetic precision/no traversal")
	}
	again, _, e := c.VirtualRouterRoutes.GetRaw(context.Background(), "id")
	v["synthetic_number"][0] = '9'
	if string(v["synthetic_number"]) != "9" {
		t.Fatal("mutation")
	}
	if e != nil || string(again["synthetic_number"]) != "0" || f.calls != 2 {
		t.Fatal("result independence")
	}
}
func TestRouterRouteRawGetBoundaries(t *testing.T) {
	f := &routeRawWire{t: t, code: 200}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	for _, id := range []string{"", "..", "a/b", "a?b", "a#b", `a\b`, "%2f", "%2e%2e", "a%3fb", ".", "é", "\n", "a\x00b"} {
		v, r, e := c.VirtualRouterRoutes.GetRaw(context.Background(), id)
		if v != nil || r != nil || e == nil || f.calls != 0 {
			t.Fatal("refusal")
		}
	}
	for _, body := range []string{``, `{`, `[]`, `1`, `true`, `"text"`, `null`, `{}`} {
		f.calls = 0
		f.data = []byte(body)
		v, r, e := c.VirtualRouterRoutes.GetRaw(context.Background(), "id")
		if body == `{}` {
			if e != nil || v == nil || len(v) != 0 || r == nil {
				t.Fatal("empty object")
			}
		} else {
			if e == nil || v != nil {
				t.Fatal("decode")
			}
			if body == `null` {
				if r == nil {
					t.Fatal("null response")
				}
			} else if r != nil {
				t.Fatal("decode response")
			}
			if body == `` && !errors.Is(e, io.EOF) {
				t.Fatal("EOF")
			}
		}
		if f.calls != 1 {
			t.Fatal("count")
		}
	}
	for _, code := range []int{403, 404, 500} {
		f.code = code
		f.calls = 0
		f.data = []byte(`[{"error_message":"denied","error_point":"uuid","error_type":"forbidden"}]`)
		v, r, e := c.VirtualRouterRoutes.GetRaw(context.Background(), "id")
		var api *ErrorResponse
		if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0] != (Error{Message: "denied", Point: "uuid", Type: "forbidden"}) || f.calls != 1 {
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
		v, r, e := c.VirtualRouterRoutes.GetRaw(ctx, "id")
		if v != nil || r != nil || !errors.Is(e, sentinel) || f.calls != 1 {
			t.Fatal("transport/context")
		}
	}
}
