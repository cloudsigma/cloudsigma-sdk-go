package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type defWire struct {
	t                            *testing.T
	method, path, body, response string
	code, calls                  int
	err                          error
}

func (f *defWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != f.method || q.URL.RequestURI() != f.path || q.URL.RawQuery != "" || q.URL.Fragment != "" {
		f.t.Fatal("route")
	}
	b, e := io.ReadAll(q.Body)
	if e != nil || !definitionJSONEqual(string(b), f.body) {
		f.t.Fatal("body mismatch (redacted)")
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.response)), Request: q}, nil
}
func TestServerDefinitionPresence(t *testing.T) {
	opts := ServerDefinition{"cpu": json.RawMessage(`0`), "auto_start": json.RawMessage(`false`), "name": json.RawMessage(`""`), "meta": json.RawMessage(`null`), "unknown": json.RawMessage(`9007199254740993`)}
	raw, _ := json.Marshal(opts)
	f := &defWire{t: t, method: "POST", path: "/api/2.0/servers/", body: `{"objects":[{"auto_start":false,"cpu":0,"meta":null,"name":"","unknown":9007199254740993}]}`, response: `{"objects":[]}`, code: 201}
	c := NewClient(NewTokenCredentialsProvider("x"), WithHTTPClient(&http.Client{Transport: f}))
	before, _ := json.Marshal(opts)
	v, r, e := c.Servers.CreateDefinition(context.Background(), []ServerDefinition{opts})
	if e != nil || r.StatusCode != 201 || v == nil || f.calls != 1 {
		t.Fatal(e)
	}
	after, _ := json.Marshal(opts)
	if !bytes.Equal(before, after) {
		t.Fatal("mutation")
	}
	_ = raw
}
func TestServerDefinitionUpdateBoundaries(t *testing.T) {
	d := ServerDefinition{"cpu": json.RawMessage(`0`), "auto_start": json.RawMessage(`false`), "unknown": json.RawMessage(`9007199254740993`)}
	for _, code := range []int{200, 403} {
		f := &defWire{t: t, method: "PUT", path: "/api/2.0/servers/id/", body: `{"auto_start":false,"cpu":0,"unknown":9007199254740993}`, response: `{}`, code: code}
		c := NewClient(NewTokenCredentialsProvider("x"), WithHTTPClient(&http.Client{Transport: f}))
		v, r, e := c.Servers.UpdateDefinition(context.Background(), "id", d)
		if code == 200 && (e != nil || v == nil || r.StatusCode != 200) {
			t.Fatal(e)
		}
		if code == 403 && (v != nil || r == nil || r.StatusCode != 403 || e == nil) {
			t.Fatal(e)
		}
	}
	for _, bad := range []string{"", "..", "a/b"} {
		f := &defWire{t: t}
		badClient := NewClient(NewTokenCredentialsProvider("x"), WithHTTPClient(&http.Client{Transport: f}))
		v, r, e := badClient.Servers.UpdateDefinition(context.Background(), bad, d)
		if v != nil || r != nil || e == nil || f.calls != 0 {
			t.Fatal("invalid")
		}
	}
}

// Decode with UseNumber: object order is irrelevant, arrays and numeric tokens are exact.
func definitionJSONEqual(a, b string) bool {
	decode := func(s string) (interface{}, error) {
		d := json.NewDecoder(strings.NewReader(s))
		d.UseNumber()
		var v interface{}
		if e := d.Decode(&v); e != nil {
			return nil, e
		}
		var extra interface{}
		if e := d.Decode(&extra); e != io.EOF {
			return nil, fmt.Errorf("trailing JSON")
		}
		return v, nil
	}
	av, ae := decode(a)
	bv, be := decode(b)
	return ae == nil && be == nil && reflect.DeepEqual(av, bv)
}
func TestDefinitionJSONOracle(t *testing.T) {
	want := `{"objects":[{"n":9007199254740993,"null":null,"f":false,"z":0,"s":"","a":[1,2]}]}`
	reordered := " {\"objects\":[{\"a\":[1,2],\"s\":\"\",\"z\":0,\"f\":false,\"null\":null,\"n\":9007199254740993}]}\n"
	if !definitionJSONEqual(want, reordered) {
		t.Fatal("order/whitespace")
	}
	for _, bad := range []string{
		strings.Replace(want, `9007199254740993`, `9007199254740992`, 1),
		strings.Replace(want, `"null":null,`, ``, 1),
		strings.Replace(want, `"f":false,`, ``, 1),
		strings.Replace(want, `"z":0,`, ``, 1),
		strings.Replace(want, `"s":"",`, ``, 1),
		strings.Replace(want, `[1,2]`, `[2,1]`, 1),
		strings.Replace(want, `false`, `"false"`, 1),
		strings.Replace(want, `[1,2]`, `{"0":1,"1":2}`, 1),
		strings.Replace(want, `"objects"`, `"object"`, 1),
		want + `{}`,
		strings.Replace(want, `[1,2]`, `[]`, 1),
		strings.Replace(want, `[1,2]`, `{}`, 1),
		`{"n":9007199254740993,"null":null,"f":false,"z":0,"s":"","a":[1,2]}`,
	} {
		if definitionJSONEqual(want, bad) {
			t.Fatalf("accepted mismatch: %s", bad)
		}
	}
}
