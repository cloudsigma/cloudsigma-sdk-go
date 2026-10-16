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

type aclListWire struct {
	t           *testing.T
	body        []byte
	code, calls int
	err         error
	query       string
}

func (f *aclListWire) path() string { return "/api/2.0/acls/" + f.query }
func (f *aclListWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "GET" || q.URL.RequestURI() != f.path() {
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
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(f.body)), Request: q}, nil
}
func aclListClient(f *aclListWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func TestACLListRawPublic(t *testing.T) {
	b, e := os.ReadFile("testdata/acl_list_response.json")
	if e != nil {
		t.Fatal(e)
	}
	f := &aclListWire{t: t, body: b, code: 200}
	c := aclListClient(f)
	got, r, e := c.ACLs.ListRaw(context.Background(), nil)
	if e != nil || r == nil || r.StatusCode != 200 || r.Meta != nil || f.calls != 1 {
		t.Fatalf("result %v", e)
	}
	encoded, _ := json.Marshal(got)
	if !reflect.DeepEqual(aclJSON(t, encoded), aclJSON(t, b)) {
		t.Fatal("full public response")
	}
	again, _, e := c.ACLs.ListRaw(context.Background(), nil)
	if e != nil || f.calls != 2 {
		t.Fatal(e)
	}
	got["objects"][0] = '!'
	if got["objects"][0] != '!' || again["objects"][0] == '!' {
		t.Fatal("ownership")
	}
	var legacy aclsRoot
	if json.Unmarshal(b, &legacy) != nil || len(legacy.ACLs) != 3 {
		t.Fatal("legacy decode")
	}
	lb, _ := json.Marshal(legacy.ACLs[0])
	var lm map[string]json.RawMessage
	_ = json.Unmarshal(lb, &lm)
	if _, ok := lm["grantees"]; ok {
		t.Fatal("reassess legacy gap")
	}
	var objects []map[string]json.RawMessage
	if json.Unmarshal(again["objects"], &objects) != nil || len(objects) != 3 || !bytes.Contains(objects[0]["grantees"], []byte("user2@example.com")) {
		t.Fatal("populated grantees")
	}

}
func TestACLListRawPresence(t *testing.T) {
	for _, s := range []string{`{}`, `{"grantees":null,"meta":{},"tags":[],"future":9007199254740993,"decimal":1.2300}`, `{"grantees":[],"rules":null}`} {
		f := &aclListWire{t: t, body: []byte(s), code: 200}
		got, _, e := aclListClient(f).ACLs.ListRaw(context.Background(), nil)
		if e != nil {
			t.Fatal(e)
		}
		b, _ := json.Marshal(got)
		if !reflect.DeepEqual(aclJSON(t, b), aclJSON(t, []byte(s))) {
			t.Fatal("presence/precision")
		}
	}
}
func TestACLListRawRefusals(t *testing.T) {
	for _, s := range []string{"null", "[]", "false", "1", "\"text\"", "{"} {
		f := &aclListWire{t: t, body: []byte(s), code: 200}
		got, _, e := aclListClient(f).ACLs.ListRaw(context.Background(), nil)
		if e == nil || got != nil || f.calls != 1 {
			t.Fatal("malformed response")
		}
	}
}
func TestACLListRawErrors(t *testing.T) {
	sentinel := errors.New("synthetic transport")
	f := &aclListWire{t: t, err: sentinel}
	got, _, e := aclListClient(f).ACLs.ListRaw(context.Background(), nil)
	if !errors.Is(e, sentinel) || got != nil || f.calls != 1 {
		t.Fatal("transport")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f = &aclListWire{t: t}
	got, _, e = aclListClient(f).ACLs.ListRaw(ctx, nil)
	if !errors.Is(e, context.Canceled) || got != nil {
		t.Fatal("cancel")
	}
	f = &aclListWire{t: t, code: 403, body: []byte(`[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`)}
	got, r, e := aclListClient(f).ACLs.ListRaw(context.Background(), nil)
	if e == nil || got != nil || r == nil || r.StatusCode != 403 || f.calls != 1 {
		t.Fatal("http error")
	}
}

func TestACLListRawTypedError(t *testing.T) {
	f := &aclListWire{t: t, code: 403, body: []byte(`[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`)}
	_, r, e := aclListClient(f).ACLs.ListRaw(context.Background(), nil)
	var api *ErrorResponse
	if !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0].Type != "forbidden" || api.Errors[0].Message != "synthetic refusal" || api.Errors[0].Point != "" {
		t.Fatal("typed fields")
	}
}

func TestACLListRawPagination(t *testing.T) {
	for _, tc := range []struct {
		opts  *ListOptions
		query string
	}{{nil, ""}, {&ListOptions{}, "?limit=0"}, {&ListOptions{Limit: 2, Offset: 3}, "?limit=2&offset=3"}} {
		f := &aclListWire{t: t, code: 200, body: []byte(`{"objects":[],"meta":{"next":"https://untrusted.invalid/"}}`), query: tc.query}
		_, r, e := aclListClient(f).ACLs.ListRaw(context.Background(), tc.opts)
		if e != nil || r.Meta != nil || f.calls != 1 {
			t.Fatal("one page", e)
		}
	}
}
