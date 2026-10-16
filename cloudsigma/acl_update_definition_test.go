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

const aclUpdateID = "adf048e8-5445-47cf-a09c-fc3cfb2c823f"

type aclUpdateWire struct {
	t           *testing.T
	want, data  []byte
	code, calls int
	err         error
	response    *http.Response
	mutate      func()
}

func (f *aclUpdateWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if f.mutate != nil {
		f.mutate()
	}
	if q.Method != "PUT" || q.URL.RequestURI() != "/api/2.0/acls/"+aclUpdateID+"/" || q.Header.Get("Content-Type") != "application/json" {
		f.t.Fatal("route/content type")
	}
	b, e := io.ReadAll(q.Body)
	if e != nil || !reflect.DeepEqual(aclJSON(f.t, b), aclJSON(f.t, f.want)) {
		f.t.Fatalf("request: %s", b)
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	f.response = &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(f.data)), Request: q}
	return f.response, nil
}
func aclUpdateClient(f *aclUpdateWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func TestACLUpdateDefinitionPublic(t *testing.T) {
	request, e := os.ReadFile("testdata/acl_update_request.json")
	if e != nil {
		t.Fatal(e)
	}
	response, e := os.ReadFile("testdata/acl_update_response.json")
	if e != nil {
		t.Fatal(e)
	}
	var in ACLDefinition
	if json.Unmarshal(request, &in) != nil {
		t.Fatal("fixture")
	}
	before, _ := json.Marshal(in)
	f := &aclUpdateWire{t: t, want: request, data: response, code: 200}
	got, r, e := aclUpdateClient(f).ACLs.UpdateDefinition(context.Background(), aclUpdateID, in)
	if e != nil || r == nil || r.Response != f.response || r.StatusCode != 200 || r.Meta != nil || f.calls != 1 {
		t.Fatalf("result %v", e)
	}
	b, _ := json.Marshal(got)
	if !reflect.DeepEqual(aclJSON(t, b), aclJSON(t, response)) {
		t.Fatal("full response")
	}
	after, _ := json.Marshal(in)
	if !bytes.Equal(before, after) {
		t.Fatal("input mutation")
	}
	again, _, e := aclUpdateClient(f).ACLs.UpdateDefinition(context.Background(), aclUpdateID, in)
	if e != nil || f.calls != 2 {
		t.Fatal(e)
	}
	got["grantees"][0] = '!'
	if got["grantees"][0] != '!' || again["grantees"][0] == '!' || in["grantees"][0] == '!' {
		t.Fatal("response ownership")
	}
	in["owner"][0] = '!'
	if in["owner"][0] != '!' || again["owner"][0] == '!' || got["owner"][0] == '!' {
		t.Fatal("input ownership")
	}
}
func TestACLUpdateDefinitionPresence(t *testing.T) {
	for _, payload := range []string{`{}`, `{"grantees":[],"rules":null,"tags":[],"meta":{},"name":"","future":9007199254740993,"false":false,"owner":null,"resource_uri":null,"uuid":null}`, `{"grantees":null,"tags":null,"rules":[],"meta":null,"future":1.2300}`} {
		var in ACLDefinition
		if json.Unmarshal([]byte(payload), &in) != nil {
			t.Fatal("input")
		}
		f := &aclUpdateWire{t: t, want: []byte(payload), data: []byte(payload), code: 200}
		got, _, e := aclUpdateClient(f).ACLs.UpdateDefinition(context.Background(), aclUpdateID, in)
		b, _ := json.Marshal(got)
		if e != nil || f.calls != 1 || !reflect.DeepEqual(aclJSON(t, b), aclJSON(t, []byte(payload))) {
			t.Fatal("presence", e)
		}
	}
	// Mutation occurs after synchronous encoding, proving request/input independence,
	// not a concurrent-mutation guarantee.
	in := ACLDefinition{"name": json.RawMessage(`"before"`)}
	f := &aclUpdateWire{t: t, want: []byte(`{"name":"before"}`), data: []byte(`{}`), code: 200, mutate: func() { in["name"][1] = 'X' }}
	_, _, e := aclUpdateClient(f).ACLs.UpdateDefinition(context.Background(), aclUpdateID, in)
	if e != nil || in["name"][1] != 'X' || f.calls != 1 {
		t.Fatal("mutation probe", e)
	}
}
func TestACLUpdateDefinitionRefusal(t *testing.T) {
	for _, id := range []string{"", ".", "..", "a/b", "a?b", "a#b", "%2f", "%2e%2e", "a\\b", "a\n", "a b", "é"} {
		f := &aclUpdateWire{t: t}
		v, r, e := aclUpdateClient(f).ACLs.UpdateDefinition(context.Background(), id, ACLDefinition{})
		if e == nil || v != nil || r != nil || f.calls != 0 {
			t.Fatal("identifier", id)
		}
	}
	for _, in := range []ACLDefinition{nil, {"bad": json.RawMessage(`{`)}, {"bad": nil}, {"bad": json.RawMessage(`1 2`)}} {
		f := &aclUpdateWire{t: t}
		v, r, e := aclUpdateClient(f).ACLs.UpdateDefinition(context.Background(), aclUpdateID, in)
		if e == nil || v != nil || r != nil || f.calls != 0 {
			t.Fatal("payload")
		}
	}
}
func TestACLUpdateDefinitionFailures(t *testing.T) {
	for _, body := range []string{"", "{", "[]", "null", "42", `"text"`} {
		f := &aclUpdateWire{t: t, want: []byte(`{}`), data: []byte(body), code: 200}
		v, r, e := aclUpdateClient(f).ACLs.UpdateDefinition(context.Background(), aclUpdateID, ACLDefinition{})
		if e == nil || v != nil || f.calls != 1 {
			t.Fatal("malformed", body)
		}
		if body == "null" && (r == nil || r.Response != f.response) {
			t.Fatal("policy response")
		}
	}
	for _, code := range []int{400, 403, 404, 500} {
		f := &aclUpdateWire{t: t, want: []byte(`{}`), data: []byte(`[{"error_type":"synthetic","error_message":"denied","error_point":null}]`), code: code}
		v, r, e := aclUpdateClient(f).ACLs.UpdateDefinition(context.Background(), aclUpdateID, ACLDefinition{})
		var api *ErrorResponse
		if !errors.As(e, &api) || v != nil || r == nil || r.Response != f.response || r.StatusCode != code || f.calls != 1 {
			t.Fatalf("HTTP error %v", e)
		}
		if api.Response != r || len(api.Errors) != 1 || api.Errors[0].Type != "synthetic" || api.Errors[0].Message != "denied" || api.Errors[0].Point != "" {
			t.Fatal("typed HTTP fields")
		}
	}
	sentinel := errors.New("synthetic transport")
	f := &aclUpdateWire{t: t, want: []byte(`{}`), err: sentinel}
	_, _, e := aclUpdateClient(f).ACLs.UpdateDefinition(context.Background(), aclUpdateID, ACLDefinition{})
	if !errors.Is(e, sentinel) || f.calls != 1 {
		t.Fatal("transport")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f = &aclUpdateWire{t: t, want: []byte(`{}`)}
	_, _, e = aclUpdateClient(f).ACLs.UpdateDefinition(ctx, aclUpdateID, ACLDefinition{})
	if !errors.Is(e, context.Canceled) || f.calls > 1 {
		t.Fatal("cancel")
	}
}
func TestACLUpdateDefinitionOracle(t *testing.T) {
	original := aclJSON(t, []byte(`{"n":9007199254740993,"a":[1,2],"null":null}`))
	for _, wrong := range []string{`{"n":9007199254740992,"a":[1,2],"null":null}`, `{"n":9007199254740993,"a":[2,1],"null":null}`, `{"n":9007199254740993,"a":[1,2]}`, `[{"n":9007199254740993,"a":[1,2],"null":null}]`} {
		if reflect.DeepEqual(original, aclJSON(t, []byte(wrong))) {
			t.Fatal("oracle negative control")
		}
	}
}
