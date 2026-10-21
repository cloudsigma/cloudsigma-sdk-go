package cloudsigma_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type remoteBulkWire struct {
	t                   *testing.T
	ids                 []string
	calls, closed, code int
	err                 error
}
type remoteBulkBody struct {
	*strings.Reader
	owner *remoteBulkWire
}

func (b *remoteBulkBody) Close() error { b.owner.closed++; return nil }
func (w *remoteBulkWire) RoundTrip(q *http.Request) (*http.Response, error) {
	w.calls++
	if q.Method != http.MethodDelete || q.URL.String() != "https://example.invalid/custom/api/2.0/remotesnapshots/" {
		w.t.Fatal("route", q.Method, q.URL)
	}
	if q.Header.Get("Authorization") != "Bearer synthetic" || q.Header.Get("Content-Type") != "application/json" {
		w.t.Fatal("headers missing")
	}
	raw, e := io.ReadAll(q.Body)
	if e != nil {
		w.t.Fatal(e)
	}
	refs := make([]map[string]string, len(w.ids))
	for i, id := range w.ids {
		refs[i] = map[string]string{"uuid": id}
	}
	want, _ := json.Marshal(map[string]interface{}{"objects": refs})
	var a, b interface{}
	if json.Unmarshal(raw, &a) != nil || json.Unmarshal(want, &b) != nil || !reflect.DeepEqual(a, b) {
		w.t.Fatal("body", string(raw))
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	if w.err != nil {
		return nil, w.err
	}
	body := ""
	if w.code >= 400 {
		body = `[{"error_message":"denied","error_point":"remote","error_type":"synthetic"}]`
	}
	return &http.Response{StatusCode: w.code, Header: make(http.Header), Body: &remoteBulkBody{strings.NewReader(body), w}, Request: q}, nil
}
func remoteBulkClient(w *remoteBulkWire) *sdk.Client {
	return sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
}
func TestRemoteSnapshotDeleteManyPublicContract(t *testing.T) {
	// UUIDs transcribed from the public multiple-remote-snapshot deletion example.
	ids := []string{"b137e217-42b6-4ecf-8575-d72efc2d3dbd", "e035a488-8587-4a15-ab25-9b7343236bc9", "feded33c-106f-49fa-a1c4-be5c718ad1b5"}
	before := append([]string(nil), ids...)
	w := &remoteBulkWire{t: t, ids: ids, code: 204}
	r, e := remoteBulkClient(w).RemoteSnapshots.DeleteMany(context.Background(), ids)
	if e != nil || r == nil || r.StatusCode != 204 || w.calls != 1 || w.closed != 1 || !reflect.DeepEqual(ids, before) {
		t.Fatal("ack/input", r, e)
	}
}
func TestRemoteSnapshotDeleteManyValidationAndOrder(t *testing.T) {
	ids := []string{"B_2", "a", "B_2"}
	w := &remoteBulkWire{t: t, ids: append([]string(nil), ids...), code: 204}
	c := remoteBulkClient(w)
	if r, e := c.RemoteSnapshots.DeleteMany(context.Background(), ids); e != nil || r == nil || !reflect.DeepEqual(ids, w.ids) {
		t.Fatal(r, e)
	}
	for _, bad := range [][]string{nil, {}, {"ok", ""}, {"ok", ".."}, {"ok", "a/b"}, {"ok", "a?b"}, {"ok", "a#b"}, {"ok", "%2f"}, {"ok", "a\\b"}, {"ok", "a\n"}, {"ok", "é"}, {"ok", "a b"}} {
		before := append([]string{}, bad...)
		r, e := c.RemoteSnapshots.DeleteMany(context.Background(), bad)
		if r != nil || e == nil || w.calls != 1 || !reflect.DeepEqual(append([]string{}, bad...), before) {
			t.Fatal("invalid dispatch/input", bad, r, e)
		}
	}
}
func TestRemoteSnapshotDeleteManyHTTPErrors(t *testing.T) {
	for _, code := range []int{401, 403, 404, 409, 429, 500} {
		w := &remoteBulkWire{t: t, ids: []string{"a"}, code: code}
		r, e := remoteBulkClient(w).RemoteSnapshots.DeleteMany(context.Background(), w.ids)
		var api *sdk.ErrorResponse
		if r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || !reflect.DeepEqual(api.Errors, []sdk.Error{{Message: "denied", Point: "remote", Type: "synthetic"}}) || w.calls != 1 || w.closed != 1 {
			t.Fatal("error fidelity", code, r, e)
		}
	}
}
func TestRemoteSnapshotDeleteManyTransportContext(t *testing.T) {
	sentinel := errors.New("synthetic transport")
	w := &remoteBulkWire{t: t, ids: []string{"a"}, code: 204, err: sentinel}
	c := remoteBulkClient(w)
	if r, e := c.RemoteSnapshots.DeleteMany(context.Background(), w.ids); r != nil || !errors.Is(e, sentinel) || w.calls != 1 {
		t.Fatal(r, e)
	}
	w.err = nil
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if r, e := c.RemoteSnapshots.DeleteMany(ctx, w.ids); r != nil || !errors.Is(e, context.Canceled) || w.calls != 2 {
		t.Fatal(r, e)
	}
	if w.closed != 0 {
		t.Fatal("unexpected response")
	}
}
