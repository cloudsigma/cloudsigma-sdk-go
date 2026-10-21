package cloudsigma_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type tagResourcesPageRawWire struct {
	t                   *testing.T
	query, body, kind   string
	code, calls, closed int
	err                 error
}
type tagResourcesPageRawBody struct {
	*strings.Reader
	wire *tagResourcesPageRawWire
}

func (b *tagResourcesPageRawBody) Close() error { b.wire.closed++; return nil }
func (w *tagResourcesPageRawWire) RoundTrip(q *http.Request) (*http.Response, error) {
	w.calls++
	if q.Body != nil {
		b, err := io.ReadAll(q.Body)
		if err != nil || len(b) != 0 {
			w.t.Fatal("unexpected request payload", string(b), err)
		}
	}
	if q.Method != "GET" || q.URL.Path != "/custom/api/2.0/tags/id/"+w.kind+"/" || q.URL.RawQuery != w.query {
		w.t.Fatalf("request: %s %s body=%v", q.Method, q.URL, q.Body)
	}
	if q.Header.Get("Authorization") != "Bearer synthetic" {
		w.t.Fatal("missing authentication")
	}
	if err := q.Context().Err(); err != nil {
		return nil, err
	}
	if w.err != nil {
		return nil, w.err
	}
	return &http.Response{StatusCode: w.code, Header: make(http.Header), Body: &tagResourcesPageRawBody{strings.NewReader(w.body), w}, Request: q}, nil
}
func tagResourcesPageRawClient(w *tagResourcesPageRawWire) *sdk.Client {
	return sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
}
func TestTagResourcesPageRawContract(t *testing.T) {
	for _, tc := range []struct {
		name  string
		opts  *sdk.ListOptions
		query string
	}{
		{"nil", nil, ""}, {"all", &sdk.ListOptions{}, "limit=0"}, {"page", &sdk.ListOptions{Limit: 2, Offset: 3}, "limit=2&offset=3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"meta":{"limit":2,"offset":3,"total_count":6,"next":"https://untrusted.invalid/"},"objects":[{"uuid":"synthetic-server","name":"tagged-server","owner":{"uuid":"owner","resource_uri":"/api/2.0/user/owner/"},"meta":{"label":"test","nullable":null},"status":"stopped","runtime":null,"resource_uri":{"resource_uri":"/api/2.0/servers/synthetic-server/","uuid":"synthetic-server"},"sequence":9007199254740993,"nullable":null,"unknown":{"number":1.2300}}],"future":null}`
			w := &tagResourcesPageRawWire{t: t, kind: "servers", query: tc.query, body: body, code: 200}
			var before sdk.ListOptions
			if tc.opts != nil {
				before = *tc.opts
			}
			got, resp, err := tagResourcesPageRawClient(w).Tags.ListResourcesPageRaw(context.Background(), "id", w.kind, tc.opts)
			if err != nil || resp == nil || resp.StatusCode != 200 || got == nil || w.calls != 1 || w.closed != 1 {
				t.Fatalf("result %v %v %v %+v", got, resp, err, w)
			}
			if string(got["objects"]) != `[{"uuid":"synthetic-server","name":"tagged-server","owner":{"uuid":"owner","resource_uri":"/api/2.0/user/owner/"},"meta":{"label":"test","nullable":null},"status":"stopped","runtime":null,"resource_uri":{"resource_uri":"/api/2.0/servers/synthetic-server/","uuid":"synthetic-server"},"sequence":9007199254740993,"nullable":null,"unknown":{"number":1.2300}}]` || string(got["future"]) != "null" || string(got["meta"]) != `{"limit":2,"offset":3,"total_count":6,"next":"https://untrusted.invalid/"}` {
				t.Fatal("raw envelope changed", got)
			}
			if tc.opts != nil && !reflect.DeepEqual(*tc.opts, before) {
				t.Fatal("mutated options")
			}
			// Each request owns its response bytes, while ordinary map copies may alias.
			got["objects"][0] = 'x'
			again, _, err := tagResourcesPageRawClient(w).Tags.ListResourcesPageRaw(context.Background(), "id", w.kind, tc.opts)
			if err != nil || again["objects"][0] != '[' || w.calls != 2 {
				t.Fatal("response ownership", err)
			}
		})
	}
}
func TestTagResourcesPageRawInvalid(t *testing.T) {
	for _, o := range []sdk.ListOptions{{Limit: -1}, {Offset: -1}, {Limit: -1, Offset: -1}} {
		w := &tagResourcesPageRawWire{t: t, kind: "servers"}
		v, r, e := tagResourcesPageRawClient(w).Tags.ListResourcesPageRaw(context.Background(), "id", w.kind, &o)
		if e == nil || v != nil || r != nil || w.calls != 0 {
			t.Fatal("invalid pagination dispatched")
		}
	}
}
func TestTagResourcesPageRawErrors(t *testing.T) {
	for _, code := range []int{401, 403, 404, 409, 429, 500} {
		w := &tagResourcesPageRawWire{t: t, kind: "servers", code: code, body: `[{"error_message":"denied","error_type":"synthetic","error_point":"tags"}]`}
		v, r, e := tagResourcesPageRawClient(w).Tags.ListResourcesPageRaw(context.Background(), "id", w.kind, nil)
		var api *sdk.ErrorResponse
		if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || w.closed != 1 {
			t.Fatalf("error %d: %v %v %v", code, v, r, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	w := &tagResourcesPageRawWire{t: t, kind: "servers", err: sentinel}
	v, r, e := tagResourcesPageRawClient(w).Tags.ListResourcesPageRaw(context.Background(), "id", w.kind, nil)
	if v != nil || r != nil || !errors.Is(e, sentinel) {
		t.Fatal("transport identity", e)
	}
	for _, body := range []string{`[]`, `{`} {
		w = &tagResourcesPageRawWire{t: t, kind: "servers", code: 200, body: body}
		v, r, e = tagResourcesPageRawClient(w).Tags.ListResourcesPageRaw(context.Background(), "id", w.kind, nil)
		if v != nil || r != nil || e == nil || w.closed != 1 {
			t.Fatal("shape accepted", body, e)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w = &tagResourcesPageRawWire{t: t, kind: "servers"}
	v, r, e = tagResourcesPageRawClient(w).Tags.ListResourcesPageRaw(ctx, "id", w.kind, nil)
	if v != nil || r != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("cancellation", e)
	}
}

var _ io.ReadCloser = (*tagResourcesPageRawBody)(nil)

func TestTagResourcesPageRawKindsAndRoutes(t *testing.T) {
	for _, kind := range []string{"servers", "drives", "ips", "vlans"} {
		w := &tagResourcesPageRawWire{t: t, kind: kind, code: 200, body: `{"objects":[],"meta":{}}`, query: "limit=1&offset=2"}
		v, r, e := tagResourcesPageRawClient(w).Tags.ListResourcesPageRaw(context.Background(), "id", kind, &sdk.ListOptions{Limit: 1, Offset: 2})
		if e != nil || r == nil || v == nil || w.calls != 1 || w.closed != 1 {
			t.Fatal(kind, v, r, e)
		}
	}
	for _, tc := range []struct{ id, kind string }{{"", "servers"}, {"../id", "servers"}, {"id?x=y", "servers"}, {"id", ""}, {"id", "unknown"}, {"id", "servers/"}} {
		w := &tagResourcesPageRawWire{t: t}
		v, r, e := tagResourcesPageRawClient(w).Tags.ListResourcesPageRaw(context.Background(), tc.id, tc.kind, nil)
		if e == nil || v != nil || r != nil || w.calls != 0 {
			t.Fatal("invalid route dispatched", tc)
		}
	}
	w := &tagResourcesPageRawWire{t: t, kind: "servers", code: 200, body: "null"}
	v, r, e := tagResourcesPageRawClient(w).Tags.ListResourcesPageRaw(context.Background(), "id", "servers", nil)
	if e == nil || v != nil || r == nil || w.closed != 1 {
		t.Fatal("null response semantics", v, r, e)
	}
}
