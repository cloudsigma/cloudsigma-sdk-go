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

type aclDefinitionWire struct {
	t           *testing.T
	want, data  []byte
	code, calls int
	err         error
}

func aclJSON(t *testing.T, b []byte) interface{} {
	t.Helper()
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	var x interface{}
	if e := d.Decode(&x); e != nil {
		t.Fatal(e)
	}
	return x
}
func (f *aclDefinitionWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/acls/" || q.Header.Get("Content-Type") != "application/json" {
		f.t.Fatal("route or content type")
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
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(f.data)), Request: q}, nil
}
func aclClient(f *aclDefinitionWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func TestACLDefinitionPublic(t *testing.T) {
	request, e := os.ReadFile("testdata/acl_create_request.json")
	if e != nil {
		t.Fatal(e)
	}
	response, e := os.ReadFile("testdata/acl_create_response.json")
	if e != nil {
		t.Fatal(e)
	}
	var in struct {
		Objects []ACLDefinition `json:"objects"`
	}
	if json.Unmarshal(request, &in) != nil {
		t.Fatal("fixture")
	}
	before, _ := json.Marshal(in)
	f := &aclDefinitionWire{t: t, want: request, data: response, code: 201}
	got, r, e := aclClient(f).ACLs.CreateDefinition(context.Background(), in.Objects)
	if e != nil || r == nil || r.StatusCode != 201 || r.Meta != nil || f.calls != 1 {
		t.Fatalf("result %v", e)
	}
	b, _ := json.Marshal(got)
	if !reflect.DeepEqual(aclJSON(t, b), aclJSON(t, response)) {
		t.Fatal("response fidelity")
	}
	after, _ := json.Marshal(in)
	if !bytes.Equal(before, after) {
		t.Fatal("input mutation")
	}
	var objects []map[string]json.RawMessage
	if json.Unmarshal(got["objects"], &objects) != nil || len(objects) != 1 || !bytes.Contains(objects[0]["grantees"], []byte("c2fc9982-cf2e-434a-bf63-e22a27b39f00")) {
		t.Fatal("grantees")
	}
	again, _, e := aclClient(f).ACLs.CreateDefinition(context.Background(), in.Objects)
	if e != nil {
		t.Fatal(e)
	}
	got["objects"][0] = '!'
	if got["objects"][0] != '!' || again["objects"][0] == '!' || response[0] != '{' {
		t.Fatal("owned response")
	}
}
func TestACLDefinitionPresence(t *testing.T) {
	for _, payload := range []string{`{"objects":[]}`, `{"objects":[{}]}`, `{"objects":[{"grantees":[],"rules":null,"tags":[],"meta":{},"name":"","future":9007199254740993,"false":false}]}`} {
		var in struct {
			Objects []ACLDefinition `json:"objects"`
		}
		if json.Unmarshal([]byte(payload), &in) != nil {
			t.Fatal("input")
		}
		f := &aclDefinitionWire{t: t, want: []byte(payload), data: []byte(`{"objects":[],"future":9007199254740993,"null":null}`), code: 201}
		got, _, e := aclClient(f).ACLs.CreateDefinition(context.Background(), in.Objects)
		if e != nil || string(got["future"]) != "9007199254740993" || string(got["null"]) != "null" || f.calls != 1 {
			t.Fatal("presence", e)
		}
	}
}
func TestACLDefinitionRefusal(t *testing.T) {
	cases := [][]ACLDefinition{nil, {nil}, {{"name": json.RawMessage(`"ok"`)}, nil}, {{"bad": json.RawMessage(`{`)}}}
	for _, key := range []string{"owner", "resource_uri", "uuid"} {
		cases = append(cases, []ACLDefinition{{"name": json.RawMessage(`"first"`)}, {key: json.RawMessage(`null`)}})
	}
	for _, in := range cases {
		f := &aclDefinitionWire{t: t}
		v, r, e := aclClient(f).ACLs.CreateDefinition(context.Background(), in)
		if e == nil || v != nil || r != nil || f.calls != 0 {
			t.Fatal("must refuse before dispatch")
		}
	}
}
func TestACLDefinitionFailures(t *testing.T) {
	for _, body := range []string{"", "{", "[]", "null"} {
		f := &aclDefinitionWire{t: t, want: []byte(`{"objects":[{}]}`), data: []byte(body), code: 201}
		v, _, e := aclClient(f).ACLs.CreateDefinition(context.Background(), []ACLDefinition{{}})
		if e == nil || v != nil || f.calls != 1 {
			t.Fatal("malformed", body)
		}
	}
	for _, code := range []int{403, 404, 500} {
		f := &aclDefinitionWire{t: t, want: []byte(`{"objects":[{}]}`), data: []byte(`[{"error_type":"synthetic","error_message":"denied","error_point":null}]`), code: code}
		v, r, e := aclClient(f).ACLs.CreateDefinition(context.Background(), []ACLDefinition{{}})
		if e == nil || v != nil || r == nil || r.StatusCode != code || f.calls != 1 {
			t.Fatal("HTTP error")
		}
	}
	sentinel := errors.New("synthetic transport")
	f := &aclDefinitionWire{t: t, want: []byte(`{"objects":[{}]}`), err: sentinel}
	_, _, e := aclClient(f).ACLs.CreateDefinition(context.Background(), []ACLDefinition{{}})
	if !errors.Is(e, sentinel) || f.calls != 1 {
		t.Fatal("transport")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f = &aclDefinitionWire{t: t, want: []byte(`{"objects":[{}]}`)}
	_, _, e = aclClient(f).ACLs.CreateDefinition(ctx, []ACLDefinition{{}})
	if !errors.Is(e, context.Canceled) {
		t.Fatal("cancel")
	}
}
