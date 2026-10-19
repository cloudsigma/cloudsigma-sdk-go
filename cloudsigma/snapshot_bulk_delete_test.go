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

type snapshotBulkWire struct {
	t           *testing.T
	want        string
	calls, code int
	closed      int
	err         error
}

func (f *snapshotBulkWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "DELETE" || q.URL.Path != "/api/2.0/snapshots/" || q.URL.RawQuery != "" || q.URL.Fragment != "" {
		f.t.Fatal("route", q.URL)
	}
	raw, e := io.ReadAll(q.Body)
	if e != nil {
		f.t.Fatal(e)
	}
	var got, want interface{}
	if e = json.Unmarshal(raw, &got); e != nil {
		f.t.Fatal(e)
	}
	if e = json.Unmarshal([]byte(f.want), &want); e != nil {
		f.t.Fatal(e)
	}
	if !reflect.DeepEqual(got, want) {
		f.t.Fatal("body", string(raw))
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	body := ""
	if f.code >= 400 {
		body = `[{"error_message":"denied","error_point":"snapshot","error_type":"synthetic"}]`
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: &snapshotBulkBody{Reader: strings.NewReader(body), owner: f}, Request: q}, nil
}

type snapshotBulkBody struct {
	*strings.Reader
	owner *snapshotBulkWire
}

func (b *snapshotBulkBody) Close() error { b.owner.closed++; return nil }

func snapshotBulkClient(f *snapshotBulkWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func TestSnapshotDeleteManyPublicFixture(t *testing.T) {
	raw, e := os.ReadFile("snapshot_bulk_delete_request.json")
	if e != nil {
		t.Fatal(e)
	}
	var body struct {
		Objects []struct {
			UUID string `json:"uuid"`
		} `json:"objects"`
	}
	if e = json.Unmarshal(raw, &body); e != nil {
		t.Fatal(e)
	}
	ids := make([]string, len(body.Objects))
	for i, v := range body.Objects {
		ids[i] = v.UUID
	}
	before := append([]string(nil), ids...)
	f := &snapshotBulkWire{t: t, want: string(raw), code: 204}
	r, e := snapshotBulkClient(f).Snapshots.DeleteMany(context.Background(), ids)
	if e != nil || r == nil || r.StatusCode != 204 || f.calls != 1 || f.closed != 1 || !reflect.DeepEqual(ids, before) {
		t.Fatal("acknowledgment/input", r, e)
	}
}
func TestSnapshotDeleteManyOrderDuplicatesAndInvalid(t *testing.T) {
	ids := []string{"b", "a", "b"}
	f := &snapshotBulkWire{t: t, want: `{"objects":[{"uuid":"b"},{"uuid":"a"},{"uuid":"b"}]}`, code: 204}
	c := snapshotBulkClient(f)
	if r, e := c.Snapshots.DeleteMany(context.Background(), ids); r == nil || e != nil || f.calls != 1 || !reflect.DeepEqual(ids, []string{"b", "a", "b"}) {
		t.Fatal("order/dedup", e)
	}
	for _, bad := range [][]string{nil, {}, {"valid", ""}, {"valid", ".."}, {"valid", "x/y"}, {"valid", "x?y"}, {"valid", "x#y"}, {"valid", "%2f"}, {"valid", "x\\y"}, {"valid", "x\n"}, {"valid", "é"}} {
		before := append([]string{}, bad...)
		r, e := c.Snapshots.DeleteMany(context.Background(), bad)
		if r != nil || e == nil || f.calls != 1 || !reflect.DeepEqual(append([]string{}, bad...), before) {
			t.Fatal("invalid dispatch/mutation", bad, r, e)
		}
	}
}
func TestSnapshotDeleteManyErrors(t *testing.T) {
	f := &snapshotBulkWire{t: t, want: `{"objects":[{"uuid":"a"}]}`, code: 403}
	c := snapshotBulkClient(f)
	ids := []string{"a"}
	for _, code := range []int{403, 404, 500} {
		f.code = code
		n := f.calls
		r, e := c.Snapshots.DeleteMany(context.Background(), ids)
		var api *ErrorResponse
		if r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || !reflect.DeepEqual(api.Errors, []Error{{Message: "denied", Point: "snapshot", Type: "synthetic"}}) || f.calls != n+1 || f.closed != f.calls {
			t.Fatal("HTTP fidelity", r, e)
		}
	}
	sentinel := errors.New("bulk transport")
	f.err = sentinel
	n := f.calls
	if r, e := c.Snapshots.DeleteMany(context.Background(), ids); r != nil || !errors.Is(e, sentinel) || f.calls != n+1 {
		t.Fatal("transport", r, e)
	}
	f.err = nil
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n = f.calls
	if r, e := c.Snapshots.DeleteMany(ctx, ids); r != nil || !errors.Is(e, context.Canceled) || f.calls != n+1 {
		t.Fatal("cancel", r, e)
	}
	if !reflect.DeepEqual(ids, []string{"a"}) {
		t.Fatal("mutation")
	}
}
