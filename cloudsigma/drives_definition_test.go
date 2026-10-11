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
	"strings"
	"testing"
)

type definitionWire struct {
	t     *testing.T
	want  DriveDefinition
	calls int
	err   error
	code  int
	body  string
}

func (f *definitionWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	if q.URL.RawQuery != "do=resize" || q.URL.Fragment != "" {
		f.t.Fatalf("query changed: %s", q.URL.String())
	}
	if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/drives/df170528-c34d-4009-a806-c7d85c78cad8/action/?do=resize" {
		f.t.Fatalf("route %s %s", q.Method, q.URL.RequestURI())
	}
	raw, e := io.ReadAll(q.Body)
	if e != nil {
		f.t.Fatal(e)
	}
	var got DriveDefinition
	if e = json.Unmarshal(raw, &got); e != nil {
		f.t.Fatal(e)
	}
	compact := func(d DriveDefinition) map[string]string {
		r := map[string]string{}
		for k, v := range d {
			var b bytes.Buffer
			if e := json.Compact(&b, v); e != nil {
				f.t.Fatal(e)
			}
			r[k] = b.String()
		}
		return r
	}
	if !reflect.DeepEqual(compact(got), compact(f.want)) {
		f.t.Fatalf("wire lost values: %s", raw)
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.body)), Request: q}, nil
}

const definitionID = "df170528-c34d-4009-a806-c7d85c78cad8"

func definitionClient(f *definitionWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func TestResizeDefinitionFullExample(t *testing.T) {
	raw, e := os.ReadFile("resize_public_request.json")
	if e != nil {
		t.Fatal(e)
	}
	response, e := os.ReadFile("resize_public_response.json")
	if e != nil {
		t.Fatal(e)
	}
	var d DriveDefinition
	if e = json.Unmarshal(raw, &d); e != nil {
		t.Fatal(e)
	}
	before, _ := json.Marshal(d)
	f := &definitionWire{t: t, want: d, code: 202, body: string(response)}
	out, r, e := definitionClient(f).Drives.ResizeDefinition(context.Background(), definitionID, d)
	if e != nil || r == nil || r.StatusCode != 202 || len(out) != 1 || out[0].Status != "resizing" || out[0].Size != 2147483648 || f.calls != 1 {
		t.Fatalf("result %v %v %v", out, r, e)
	}
	after, _ := json.Marshal(d)
	if !bytes.Equal(before, after) {
		t.Fatal("caller mutation")
	}
	for _, k := range []string{"affinities", "grantees", "jobs", "permissions", "allow_multimount", "licenses", "meta", "mounted_on", "snapshots", "tags"} {
		if _, ok := d[k]; !ok {
			t.Fatal("fixture lacks", k)
		}
	}
}
func TestResizeDefinitionValuesAndLicense(t *testing.T) {
	var d DriveDefinition
	if err := json.Unmarshal([]byte(`{"unknown":9007199254740993,"false":false,"zero":0,"empty":[],"object":{},"null":null,"runtime":{"is_snapshotable":null,"snapshots_allocated_size":0}}`), &d); err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(d)
	f := &definitionWire{t: t, want: d, code: 202, body: `{"objects":[{"uuid":"synthetic","licenses":[{"license":{"free":0}}]}]}`}
	out, _, e := definitionClient(f).Drives.ResizeDefinition(context.Background(), definitionID, d)
	if e != nil || out[0].Licenses[0].License.Free != "0" {
		t.Fatal("private decode", e)
	}
	after, _ := json.Marshal(d)
	if !bytes.Equal(before, after) {
		t.Fatal("mutation")
	}
}
func TestResizeDefinitionInvalidNoDispatch(t *testing.T) {
	f := &definitionWire{t: t}
	c := definitionClient(f)
	for _, raw := range []string{`[]`, `1`, `"text"`, `{bad`} {
		var d DriveDefinition
		if json.Unmarshal([]byte(raw), &d) == nil {
			t.Fatal("nonobject accepted", raw)
		}
	}
	for _, d := range []DriveDefinition{nil, {"bad": json.RawMessage(`{`)}, {"nilraw": nil}} {
		if _, _, e := c.Drives.ResizeDefinition(context.Background(), definitionID, d); e == nil {
			t.Fatal("invalid accepted")
		}
	}
	if _, _, e := c.Drives.ResizeDefinition(context.Background(), "", DriveDefinition{}); !errors.Is(e, ErrEmptyArgument) {
		t.Fatal(e)
	}
	if f.calls != 0 {
		t.Fatal("invalid dispatched")
	}
}
func TestResizeDefinitionErrorsContextOrigin(t *testing.T) {
	sentinel := errors.New("synthetic transport")
	f := &definitionWire{t: t, err: sentinel}
	c := definitionClient(f)
	d := DriveDefinition{}
	if _, _, e := c.Drives.ResizeDefinition(context.Background(), definitionID, d); !errors.Is(e, sentinel) {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f.err = nil
	if _, _, e := c.Drives.ResizeDefinition(ctx, definitionID, d); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	calls := f.calls
	if _, e := c.NewRequest("POST", "https://off-origin.invalid/api/2.0/drives/id/action/?do=resize", d); e == nil {
		t.Fatal("off-origin accepted")
	}
	if f.calls != calls {
		t.Fatal("origin dispatched")
	}
	f.want = d
	f.code = 400
	f.body = `[]`
	out, r, e := c.Drives.ResizeDefinition(context.Background(), definitionID, d)
	if e == nil || out != nil || r == nil || r.StatusCode != 400 {
		t.Fatal("HTTP error lost", e)
	}
	f.code = 202
	f.body = `{`
	if _, r, e := c.Drives.ResizeDefinition(context.Background(), definitionID, d); e == nil || r != nil {
		t.Fatal("existing nil-response decode error convention changed")
	}
}

// All values are local synthetic input, never public API probes.
func TestResizeDefinitionIdentifierNoDispatch(t *testing.T) {
	invalid := []string{"", ".", "..", "id?do=delete&x=", "../../servers/target", "id/path", `id\path`, "id#frag", "%2f", "%2F", "%5c", "%2e%2e", ".%2E/servers/target", "%252f", "id%3fdo=delete", "%00", "%0a", "%", "id name", " id", "id ", "id\t", "id\n", "id\r", "id\x00", "id\x7f", "id\u00a0", "idé", "id:port", "id@host", "id&x=y", "id+name", "id.name"}
	for _, id := range invalid {
		t.Run(id, func(t *testing.T) {
			f := &definitionWire{t: t}
			out, resp, err := definitionClient(f).Drives.ResizeDefinition(context.Background(), id, DriveDefinition{})
			if err == nil || out != nil || resp != nil || f.calls != 0 {
				t.Fatalf("invalid ID dispatched: %q calls=%d error=%v", id, f.calls, err)
			}
		})
	}
}
