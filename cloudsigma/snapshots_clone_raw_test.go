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
	"time"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type snapshotsCloneRawWire struct {
	t                   *testing.T
	body, want          string
	code, calls, closed int
	err                 error
}
type snapshotsCloneRawBody struct {
	*strings.Reader
	w *snapshotsCloneRawWire
}

func (b *snapshotsCloneRawBody) Close() error { b.w.closed++; return nil }
func (w *snapshotsCloneRawWire) RoundTrip(q *http.Request) (*http.Response, error) {
	w.calls++
	if q.Method != "POST" || q.URL.RequestURI() != "/custom/api/2.0/snapshots/snapshot-1/action/?do=clone" {
		w.t.Fatalf("route %s %s", q.Method, q.URL)
	}
	if q.Header.Get("Authorization") != "Bearer synthetic" || q.Header.Get("Content-Type") != "application/json" || q.Header.Get("Accept") != "application/json" {
		w.t.Fatal("headers")
	}
	b, e := io.ReadAll(q.Body)
	if e != nil {
		w.t.Fatal(e)
	}
	var got, want map[string]string
	if json.Unmarshal(b, &got) != nil || json.Unmarshal([]byte(w.want), &want) != nil || !reflect.DeepEqual(got, want) {
		w.t.Fatalf("body %s wanted %s", b, w.want)
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if w.err != nil {
		return nil, w.err
	}
	return &http.Response{StatusCode: w.code, Header: make(http.Header), Body: &snapshotsCloneRawBody{strings.NewReader(w.body), w}, Request: q}, nil
}
func snapshotsCloneRawClient(w *snapshotsCloneRawWire) *sdk.Client {
	return sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
}
func TestSnapshotsCloneRawSuccess(t *testing.T) {
	w := &snapshotsCloneRawWire{t: t, code: 202, want: `{}`, body: `{"uuid":"drive-1","status":"cloning_dst","jobs":[{"uuid":"job-1","resource_uri":"/api/2.0/jobs/job-1/"}],"resource_uri":{"uuid":"drive-1","resource_uri":"/api/2.0/drives/drive-1/"},"runtime":{"is_snapshotable":null,"snapshots_allocated_size":0,"storage_type":"dssd"},"meta":{},"tags":[],"future":{"n":123456789012345678901},"nullable":null}`}
	c := snapshotsCloneRawClient(w)
	v, r, e := c.Snapshots.CloneRaw(context.Background(), "snapshot-1")
	if e != nil || r == nil || r.StatusCode != 202 || r.Meta != nil || w.calls != 1 || w.closed != 1 {
		t.Fatalf("result %v %v %v", v, r, e)
	}
	var want map[string]json.RawMessage
	if json.Unmarshal([]byte(w.body), &want) != nil || !reflect.DeepEqual(v, want) {
		t.Fatal("raw envelope", v)
	}
	v["future"][0] = '['
	v2, _, e := c.Snapshots.CloneRaw(context.Background(), "snapshot-1")
	if e != nil || !reflect.DeepEqual(v2, want) || w.calls != 2 || w.closed != 2 {
		t.Fatal("independent result", e)
	}
	w.body = `{}`
	v, _, e = c.Snapshots.CloneRaw(context.Background(), "snapshot-1")
	if e != nil || v == nil || len(v) != 0 {
		t.Fatal("empty raw object", e)
	}
}
func TestSnapshotsCloneRawRefusals(t *testing.T) {
	for _, id := range []string{"", ".", "..", "a/b", "a?x", "a#x", "%2f", "a\\b", "a\n", "a\x00"} {
		w := &snapshotsCloneRawWire{t: t}
		v, r, e := snapshotsCloneRawClient(w).Snapshots.CloneRaw(context.Background(), id)
		if e == nil || v != nil || r != nil || w.calls != 0 {
			t.Fatalf("identifier %q", id)
		}
	}
	for _, endpoint := range []string{"https://example.invalid/custom/api/2.0/", "not-an-endpoint"} {
		w := &snapshotsCloneRawWire{t: t}
		c := sdk.NewClient(nil, sdk.WithEndpoint(endpoint), sdk.WithHTTPClient(&http.Client{Transport: w}))
		v, r, e := c.Snapshots.CloneRaw(context.Background(), "snapshot-1")
		if e == nil || v != nil || r != nil || w.calls != 0 {
			t.Fatal("request setup")
		}
	}
}
func TestSnapshotsCloneRawErrors(t *testing.T) {
	for _, code := range []int{400, 401, 403, 404, 409, 429, 500} {
		w := &snapshotsCloneRawWire{t: t, code: code, want: `{}`, body: `[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`}
		v, r, e := snapshotsCloneRawClient(w).Snapshots.CloneRaw(context.Background(), "snapshot-1")
		var api *sdk.ErrorResponse
		if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0].Message != "synthetic refusal" || w.calls != 1 || w.closed != 1 {
			t.Fatalf("HTTP%d %v", code, e)
		}
	}
	for _, body := range []string{`[]`, `null`, `{"name":`, ``} {
		w := &snapshotsCloneRawWire{t: t, code: 202, want: `{}`, body: body}
		v, r, e := snapshotsCloneRawClient(w).Snapshots.CloneRaw(context.Background(), "snapshot-1")
		if v != nil || e == nil || w.calls != 1 || w.closed != 1 || (body == "null" && r == nil) || (body != "null" && r != nil) {
			t.Fatalf("malformed %q %v", body, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	w := &snapshotsCloneRawWire{t: t, err: sentinel, want: `{}`}
	v, r, e := snapshotsCloneRawClient(w).Snapshots.CloneRaw(context.Background(), "snapshot-1")
	if v != nil || r != nil || !errors.Is(e, sentinel) || w.calls != 1 || w.closed != 0 {
		t.Fatal("transport", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w = &snapshotsCloneRawWire{t: t, want: `{}`}
	v, r, e = snapshotsCloneRawClient(w).Snapshots.CloneRaw(ctx, "snapshot-1")
	if v != nil || r != nil || !errors.Is(e, context.Canceled) || w.calls > 1 || w.closed != 0 {
		t.Fatal("context", e)
	}
}

func TestSnapshotsCloneRawDeadlineAndUncertainCommit(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(1, 0))
	defer cancel()
	w := &snapshotsCloneRawWire{t: t, want: `{}`}
	v, r, e := snapshotsCloneRawClient(w).Snapshots.CloneRaw(ctx, "snapshot-1")
	if v != nil || r != nil || !errors.Is(e, context.DeadlineExceeded) || w.calls > 1 || w.closed != 0 {
		t.Fatal("deadline", e)
	}
	// The transport may lose the reply after the remote action commits. Never replay.
	uncertain := errors.New("reply lost after possible remote commit")
	w = &snapshotsCloneRawWire{t: t, want: `{}`, err: uncertain}
	v, r, e = snapshotsCloneRawClient(w).Snapshots.CloneRaw(context.Background(), "snapshot-1")
	if v != nil || r != nil || !errors.Is(e, uncertain) || w.calls != 1 || w.closed != 0 {
		t.Fatal("uncertain outcome must not replay", e)
	}
}
