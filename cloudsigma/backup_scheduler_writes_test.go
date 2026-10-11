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

type schedulerWriteWire struct {
	t           *testing.T
	method      string
	want        BackupScheduler
	body        string
	code, calls int
	err         error
}

func (f *schedulerWriteWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.URL.RawQuery != "" || q.URL.Fragment != "" {
		f.t.Fatal("query or fragment")
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	if q.Method != f.method || q.URL.RequestURI() != "/api/2.0/backupschedulers/1c232887-aa47-45a8-96e6-3293f3b220cb/" {
		f.t.Fatal("route", q.Method, q.URL)
	}
	if f.method == "PUT" {
		var got BackupScheduler
		if e := json.NewDecoder(q.Body).Decode(&got); e != nil {
			f.t.Fatal(e)
		}
		compact := func(d BackupScheduler) map[string]string {
			m := map[string]string{}
			for k, v := range d {
				var b bytes.Buffer
				if e := json.Compact(&b, v); e != nil {
					f.t.Fatal(e)
				}
				m[k] = b.String()
			}
			return m
		}
		if !reflect.DeepEqual(compact(got), compact(f.want)) {
			f.t.Fatal("payload changed")
		}
	} else if q.Body != nil {
		b, e := io.ReadAll(q.Body)
		if e != nil || len(b) != 0 {
			f.t.Fatal("delete body", e)
		}
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.body)), Request: q}, nil
}
func writeClient(f *schedulerWriteWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}

const schedulerWriteUUID = "1c232887-aa47-45a8-96e6-3293f3b220cb"

func TestSchedulerUpdatePublicExample(t *testing.T) {
	raw, e := os.ReadFile("scheduler_update_request.json")
	if e != nil {
		t.Fatal(e)
	}
	response, e := os.ReadFile("scheduler_update_response.json")
	if e != nil {
		t.Fatal(e)
	}
	var d BackupScheduler
	if e = json.Unmarshal(raw, &d); e != nil {
		t.Fatal(e)
	}
	before, e := json.Marshal(d)
	if e != nil {
		t.Fatal(e)
	}
	f := &schedulerWriteWire{t: t, method: "PUT", want: d, body: string(response), code: 200}
	out, r, e := writeClient(f).BackupSchedulers.Update(context.Background(), schedulerWriteUUID, d)
	if e != nil || r == nil || r.StatusCode != 200 || f.calls != 1 || string(out["is_default"]) != "false" || string(out["uuid"]) != `"ead05317-e67f-4603-9afe-4cb6aab52e52"` {
		t.Fatal("response", out, e)
	}
	var expected BackupScheduler
	if e = json.Unmarshal(response, &expected); e != nil {
		t.Fatal(e)
	}
	canonical := func(v BackupScheduler) string {
		b, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		return string(b)
	}
	if canonical(out) != canonical(expected) {
		t.Fatal("complete response mismatch")
	}
	after, e := json.Marshal(d)
	if e != nil || !bytes.Equal(before, after) {
		t.Fatal("mutation", e)
	}
}
func TestSchedulerUpdatePresenceAndFailures(t *testing.T) {
	d := BackupScheduler{"false": json.RawMessage(`false`), "zero": json.RawMessage(`0`), "empty": json.RawMessage(`[]`), "object": json.RawMessage(`{}`), "null": json.RawMessage(`null`), "unknown": json.RawMessage(`9007199254740993`)}
	f := &schedulerWriteWire{t: t, method: "PUT", want: d, code: 200, body: `{"unknown":9007199254740993}`}
	c := writeClient(f)
	out, _, e := c.BackupSchedulers.Update(context.Background(), schedulerWriteUUID, d)
	if e != nil || string(out["unknown"]) != "9007199254740993" {
		t.Fatal(e)
	}
	n := f.calls
	for _, id := range []string{"", "..", "x?do=delete", "%2f", "x/y", "x\\y", "x\n"} {
		if _, _, e = c.BackupSchedulers.Update(context.Background(), id, d); e == nil {
			t.Fatal("unsafe accepted")
		}
	}
	for _, bad := range []BackupScheduler{nil, {"bad": json.RawMessage(`{`)}} {
		if _, _, e = c.BackupSchedulers.Update(context.Background(), schedulerWriteUUID, bad); e == nil {
			t.Fatal("bad JSON")
		}
	}
	if f.calls != n {
		t.Fatal("invalid dispatch")
	}
	sentinel := errors.New("synthetic")
	f.err = sentinel
	if _, _, e = c.BackupSchedulers.Update(context.Background(), schedulerWriteUUID, d); !errors.Is(e, sentinel) {
		t.Fatal(e)
	}
	f.err = nil
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, e = c.BackupSchedulers.Update(ctx, schedulerWriteUUID, d); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	f.code = 403
	f.body = `[]`
	if out, r, e := c.BackupSchedulers.Update(context.Background(), schedulerWriteUUID, d); out != nil || r == nil || e == nil {
		t.Fatal("error lost")
	}
	f.code = 200
	f.body = `{`
	if _, _, e = c.BackupSchedulers.Update(context.Background(), schedulerWriteUUID, d); e == nil {
		t.Fatal("bad response")
	}
}
func TestSchedulerDeleteAcceptedNotCompletion(t *testing.T) {
	f := &schedulerWriteWire{t: t, method: "DELETE", code: 204}
	c := writeClient(f)
	r, e := c.BackupSchedulers.Delete(context.Background(), schedulerWriteUUID)
	if e != nil || r == nil || r.StatusCode != 204 || f.calls != 1 {
		t.Fatal("delete", e)
	}
	n := f.calls
	for _, id := range []string{"", "..", "x?y", "%2f", "x/y"} {
		if _, e = c.BackupSchedulers.Delete(context.Background(), id); e == nil {
			t.Fatal("unsafe")
		}
	}
	if f.calls != n {
		t.Fatal("invalid dispatch")
	}
	f.code = 404
	f.body = `[]`
	if r, e = c.BackupSchedulers.Delete(context.Background(), schedulerWriteUUID); r == nil || e == nil || r.StatusCode != 404 {
		t.Fatal("404 not preserved")
	}
}

func TestSchedulerWriteRepairBoundaries(t *testing.T) {
	d := BackupScheduler{"name": json.RawMessage(`"test"`)}
	f := &schedulerWriteWire{t: t, method: "PUT", want: d, code: 200, body: `{}`}
	c := writeClient(f)
	for _, id := range []string{"", "..", "x/y", "x?y", "x#y", "%2f", "x\\y", "x\n"} {
		if v, r, e := c.BackupSchedulers.Update(context.Background(), id, d); v != nil || r != nil || e == nil {
			t.Fatal("update invalid returns")
		}
		if r, e := c.BackupSchedulers.Delete(context.Background(), id); r != nil || e == nil {
			t.Fatal("delete invalid returns")
		}
	}
	for _, bad := range []BackupScheduler{nil, {"x": nil}, {"x": json.RawMessage{}}, {"x": json.RawMessage(`{`)}} {
		if v, r, e := c.BackupSchedulers.Update(context.Background(), schedulerWriteUUID, bad); v != nil || r != nil || e == nil {
			t.Fatal("raw invalid returns")
		}
	}
	if f.calls != 0 {
		t.Fatal("invalid input dispatch")
	}
	for _, raw := range []string{`[]`, `1`, `"scalar"`} {
		var v BackupScheduler
		if json.Unmarshal([]byte(raw), &v) == nil {
			t.Fatal("nonobject accepted by Go map decoder")
		}
	}
	for _, tc := range []struct {
		body        string
		bad, nilMap bool
		keys        int
	}{{"", true, true, 0}, {`null`, false, true, 0}, {`{}`, false, false, 0}, {`[]`, true, true, 0}, {`1`, true, true, 0}, {`{`, true, true, 0}, {`{"objects":[]}`, false, false, 1}} {
		f.body = tc.body
		n := f.calls
		v, r, e := c.BackupSchedulers.Update(context.Background(), schedulerWriteUUID, d)
		if f.calls != n+1 {
			t.Fatal("followup request")
		}
		if tc.bad {
			if e == nil || v != nil || r != nil {
				t.Fatal("decode error convention", tc.body, v, r, e)
			}
		} else if e != nil || r == nil || r.StatusCode != 200 || (v == nil) != tc.nilMap || len(v) != tc.keys {
			t.Fatal("response characterization", tc.body, v, r, e)
		}
	}
	f.code = 403
	f.body = `[]`
	n := f.calls
	if v, r, e := c.BackupSchedulers.Update(context.Background(), schedulerWriteUUID, d); v != nil || r == nil || r.StatusCode != 403 || e == nil {
		t.Fatal("403 convention")
	}
	if f.calls != n+1 {
		t.Fatal("followup")
	}
	f.method = "DELETE"
	f.code = 404
	if r, e := c.BackupSchedulers.Delete(context.Background(), schedulerWriteUUID); r == nil || r.StatusCode != 404 || e == nil {
		t.Fatal("404 convention")
	}
	if f.calls != n+2 {
		t.Fatal("delete followup")
	}
	sentinel := errors.New("delete sentinel")
	f.err = sentinel
	n = f.calls
	if r, e := c.BackupSchedulers.Delete(context.Background(), schedulerWriteUUID); r != nil || !errors.Is(e, sentinel) {
		t.Fatal("delete transport", r, e)
	}
	if f.calls != n+1 {
		t.Fatal("transport followup")
	}
	f.err = nil
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n = f.calls
	if r, e := c.BackupSchedulers.Delete(ctx, schedulerWriteUUID); r != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("delete cancellation", r, e)
	}
	if f.calls != n+1 {
		t.Fatal("cancellation transport count")
	}
}

func TestSchedulerWriteExactHTTPErrors(t *testing.T) {
	for _, method := range []string{"PUT", "DELETE"} {
		code := 403
		if method == "DELETE" {
			code = 404
		}
		d := BackupScheduler{"name": json.RawMessage(`"test"`)}
		f := &schedulerWriteWire{t: t, method: method, want: d, code: code, body: `[{"error_message":"denied","error_point":"scheduler","error_type":"synthetic"}]`}
		c := writeClient(f)
		var r *Response
		var e error
		if method == "PUT" {
			var v BackupScheduler
			v, r, e = c.BackupSchedulers.Update(context.Background(), schedulerWriteUUID, d)
			if v != nil {
				t.Fatal("error value")
			}
		} else {
			r, e = c.BackupSchedulers.Delete(context.Background(), schedulerWriteUUID)
		}
		var api *ErrorResponse
		if r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || !reflect.DeepEqual(api.Errors, []Error{{Message: "denied", Point: "scheduler", Type: "synthetic"}}) || f.calls != 1 {
			t.Fatal("exact HTTP error", r, e)
		}
	}
}
