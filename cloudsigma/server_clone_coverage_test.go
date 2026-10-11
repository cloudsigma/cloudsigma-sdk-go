package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type cloneCoverageWire struct {
	t        *testing.T
	want     ServerCloneRequest
	calls    int
	mutate   func()
	response string
}

func (f *cloneCoverageWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/servers/"+cloneID+"/action/?do=clone" || q.URL.Fragment != "" {
		f.t.Fatal("route")
	}
	// Synchronous callback occurs after Clone has snapshotted and encoded input.
	// No concurrent mutation or race is introduced.
	if f.mutate != nil {
		f.mutate()
	}
	var raw []byte
	var e error
	if q.Body != nil {
		raw, e = io.ReadAll(q.Body)
	}
	if e != nil {
		f.t.Fatal(e)
	}
	if f.want == nil {
		if len(raw) != 0 {
			f.t.Fatal("nil options must have zero wire bytes, not JSON null")
		}
	} else {
		var got ServerCloneRequest
		if e = json.Unmarshal(raw, &got); e != nil {
			f.t.Fatal(e)
		}
		if !reflect.DeepEqual(got, f.want) {
			f.t.Fatal("raw precision/presence changed")
		}
	}
	return &http.Response{StatusCode: 202, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.response)), Request: q}, nil
}
func cloneCoverageClient(f *cloneCoverageWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func TestServerCloneIndividualOptionsAndPrecision(t *testing.T) {
	for _, options := range []ServerCloneRequest{nil, {}, {"name": json.RawMessage(`"only-name"`)}, {"random_vnc_password": json.RawMessage(`true`)}, {"random_vnc_password": json.RawMessage(`false`)}, {"large": json.RawMessage(`9007199254740993123456789`), "nested": json.RawMessage(`{"empty":[],"flag":false,"zero":0,"null":null}`)}} {
		before := make(ServerCloneRequest, len(options))
		for k, v := range options {
			before[k] = append(json.RawMessage(nil), v...)
		}
		f := &cloneCoverageWire{t: t, want: options, response: `{}`}
		v, r, e := cloneCoverageClient(f).Servers.Clone(context.Background(), cloneID, options)
		if e != nil || v == nil || r == nil || r.StatusCode != 202 || f.calls != 1 {
			t.Fatal("request")
		}
		if len(options) != len(before) {
			t.Fatal("map mutation")
		}
		for k, raw := range before {
			if !bytes.Equal(options[k], raw) {
				t.Fatal("caller byte mutation")
			}
		}
	}
}
func TestServerCloneSnapshotAfterEncoding(t *testing.T) {
	options := ServerCloneRequest{"name": json.RawMessage(`"before"`), "random_vnc_password": json.RawMessage(`false`)}
	want := ServerCloneRequest{"name": json.RawMessage(`"before"`), "random_vnc_password": json.RawMessage(`false`)}
	f := &cloneCoverageWire{t: t, want: want, response: `{}`, mutate: func() {
		copy(options["name"], []byte(`"after!"`))
		delete(options, "random_vnc_password")
		options["added"] = json.RawMessage(`true`)
	}}
	_, _, e := cloneCoverageClient(f).Servers.Clone(context.Background(), cloneID, options)
	if e != nil || f.calls != 1 {
		t.Fatal("snapshot")
	}
	if string(options["name"]) != `"after!"` || len(options) != 2 {
		t.Fatal("callback did not run")
	}
}
func TestServerCloneNestedLicenseCompatibility(t *testing.T) {
	// Synthetic SDK decoder compatibility fixture, NOT a public server guarantee.
	f := &cloneCoverageWire{t: t, response: `{"uuid":"synthetic-clone","drives":[{"drive":{"licenses":[{"license":{"free":9007199254740993,"initial_purchase_requirement":0,"subsequent_purchase_requirement":2.5}}]}}]}`}
	v, r, e := cloneCoverageClient(f).Servers.Clone(context.Background(), cloneID, nil)
	if e != nil || r == nil || r.StatusCode != 202 || f.calls != 1 || v == nil || len(v.Drives) != 1 || v.Drives[0].Drive == nil || len(v.Drives[0].Drive.Licenses) != 1 || v.Drives[0].Drive.Licenses[0].License == nil {
		t.Fatal("nested decode")
	}
	l := v.Drives[0].Drive.Licenses[0].License
	if l.Free != "9007199254740993" || l.InitialPurchaseRequirement != "0" || l.SubsequentPurchaseRequirement != "2.5" {
		t.Fatal("numeric license precision")
	}
}
