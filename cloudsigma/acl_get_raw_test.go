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

type aclGetWire struct {
	t           *testing.T
	body        []byte
	code, calls int
	err         error
}

func (f *aclGetWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/acls/"+aclUpdateID+"/" {
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
func aclGetClient(f *aclGetWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func TestACLGetRawPublic(t *testing.T) {
	b, e := os.ReadFile("testdata/acl_get_response.json")
	if e != nil {
		t.Fatal(e)
	}
	f := &aclGetWire{t: t, body: b, code: 200}
	c := aclGetClient(f)
	got, r, e := c.ACLs.GetRaw(context.Background(), aclUpdateID)
	if e != nil || r == nil || r.StatusCode != 200 || r.Meta != nil || f.calls != 1 {
		t.Fatalf("result %v", e)
	}
	encoded, _ := json.Marshal(got)
	if !reflect.DeepEqual(aclJSON(t, encoded), aclJSON(t, b)) {
		t.Fatal("full public response")
	}
	again, _, e := c.ACLs.GetRaw(context.Background(), aclUpdateID)
	if e != nil || f.calls != 2 {
		t.Fatal(e)
	}
	got["grantees"][0] = '!'
	if got["grantees"][0] != '!' || again["grantees"][0] == '!' {
		t.Fatal("ownership")
	}
	// The same documented fixture loses grantees in the unchanged typed model.
	var legacy ACL
	if json.Unmarshal(b, &legacy) != nil {
		t.Fatal("legacy decode")
	}
	lb, _ := json.Marshal(legacy)
	var lm map[string]json.RawMessage
	_ = json.Unmarshal(lb, &lm)
	if _, ok := lm["grantees"]; ok {
		t.Fatal("reassess legacy gap")
	}
}
func TestACLGetRawPresence(t *testing.T) {
	for _, s := range []string{`{}`, `{"grantees":null,"meta":{},"tags":[],"future":9007199254740993,"decimal":1.2300}`, `{"grantees":[],"rules":null}`} {
		f := &aclGetWire{t: t, body: []byte(s), code: 200}
		got, _, e := aclGetClient(f).ACLs.GetRaw(context.Background(), aclUpdateID)
		if e != nil {
			t.Fatal(e)
		}
		b, _ := json.Marshal(got)
		if !reflect.DeepEqual(aclJSON(t, b), aclJSON(t, []byte(s))) {
			t.Fatal("presence/precision")
		}
	}
}
func TestACLGetRawRefusals(t *testing.T) {
	for _, id := range []string{"", "..", "a/b", "a?x", "a#x", "%2f", "a\n"} {
		f := &aclGetWire{t: t}
		got, r, e := aclGetClient(f).ACLs.GetRaw(context.Background(), id)
		if e == nil || got != nil || r != nil || f.calls != 0 {
			t.Fatal("identifier dispatch")
		}
	}
	for _, s := range []string{"null", "[]", "false", "1", "\"text\"", "{"} {
		f := &aclGetWire{t: t, body: []byte(s), code: 200}
		got, _, e := aclGetClient(f).ACLs.GetRaw(context.Background(), aclUpdateID)
		if e == nil || got != nil || f.calls != 1 {
			t.Fatal("malformed response")
		}
	}
}
func TestACLGetRawErrors(t *testing.T) {
	sentinel := errors.New("synthetic transport")
	f := &aclGetWire{t: t, err: sentinel}
	got, _, e := aclGetClient(f).ACLs.GetRaw(context.Background(), aclUpdateID)
	if !errors.Is(e, sentinel) || got != nil || f.calls != 1 {
		t.Fatal("transport")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f = &aclGetWire{t: t}
	got, _, e = aclGetClient(f).ACLs.GetRaw(ctx, aclUpdateID)
	if !errors.Is(e, context.Canceled) || got != nil {
		t.Fatal("cancel")
	}
	f = &aclGetWire{t: t, code: 403, body: []byte(`[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`)}
	got, r, e := aclGetClient(f).ACLs.GetRaw(context.Background(), aclUpdateID)
	if e == nil || got != nil || r == nil || r.StatusCode != 403 || f.calls != 1 {
		t.Fatal("http error")
	}
}

func TestACLGetRawTypedError(t *testing.T) {
	f := &aclGetWire{t: t, code: 403, body: []byte(`[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`)}
	_, r, e := aclGetClient(f).ACLs.GetRaw(context.Background(), aclUpdateID)
	var api *ErrorResponse
	if !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0].Type != "forbidden" || api.Errors[0].Message != "synthetic refusal" || api.Errors[0].Point != "" {
		t.Fatal("typed fields")
	}
}
