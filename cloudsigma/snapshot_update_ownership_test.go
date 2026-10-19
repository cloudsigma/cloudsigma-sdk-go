package cloudsigma

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type snapshotUpdateWire struct {
	t                   *testing.T
	calls, closed, code int
	body                string
	err                 error
}

func (f *snapshotUpdateWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "PUT" || q.URL.Path != "/api/2.0/snapshots/target-id/" || q.URL.RawQuery != "" || q.URL.Fragment != "" || q.Header.Get("Content-Type") != "application/json" {
		f.t.Fatal("request", q)
	}
	raw, e := io.ReadAll(q.Body)
	if e != nil {
		f.t.Fatal(e)
	}
	var got map[string]json.RawMessage
	if e = json.Unmarshal(raw, &got); e != nil {
		f.t.Fatal(e)
	}
	// Independent legacy wire expectation: UUID omitted, nil Tags normalized.
	if len(got) != 3 || string(got["name"]) != `"renamed"` || string(got["tags"]) != "[]" || string(got["meta"]) != `{"key":"value"}` {
		f.t.Fatal(string(raw))
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Request: q, Body: &snapshotUpdateBody{strings.NewReader(f.body), f}}, nil
}

type snapshotUpdateBody struct {
	*strings.Reader
	owner *snapshotUpdateWire
}

func (b *snapshotUpdateBody) Close() error { b.owner.closed++; return nil }
func snapshotUpdateClient(f *snapshotUpdateWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}

func TestSnapshotUpdateOwnership(t *testing.T) {
	sentinel := errors.New("synthetic transport")
	for _, tc := range []struct {
		name   string
		code   int
		body   string
		err    error
		cancel bool
	}{
		{"success", 200, `{"uuid":"target-id","name":"renamed","tags":[]}`, nil, false},
		{"http-error", 403, `[{"error_type":"denied","error_message":"synthetic"}]`, nil, false},
		{"malformed", 200, `{`, nil, false},
		{"transport", 0, "", sentinel, false},
		{"cancelled", 0, "", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &snapshotUpdateWire{t: t, code: tc.code, body: tc.body, err: tc.err}
			input := &SnapshotUpdateRequest{Snapshot: &Snapshot{UUID: "caller-id", Name: "renamed", Meta: map[string]interface{}{"key": "value"}}}
			original := input.Snapshot
			// Do not marshal the input to snapshot it: the legacy marshaler mutates nil Tags.
			want := &Snapshot{UUID: "caller-id", Name: "renamed", Meta: map[string]interface{}{"key": "value"}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancel {
				cancel()
			}
			got, resp, e := snapshotUpdateClient(f).Snapshots.Update(ctx, "target-id", input)
			if input.Snapshot != original || !reflect.DeepEqual(input.Snapshot, want) {
				t.Errorf("caller modified: %#v", input.Snapshot)
			}
			if f.calls != 1 {
				t.Fatal("dispatch", f.calls)
			}
			if tc.name == "success" {
				if e != nil || got == nil || got.UUID != "target-id" || resp == nil || resp.StatusCode != 200 {
					t.Fatal(got, resp, e)
				}
			} else if e == nil {
				t.Fatal("expected error")
			}
			if tc.err != nil && !errors.Is(e, sentinel) {
				t.Fatal(e)
			}
			if tc.cancel && !errors.Is(e, context.Canceled) {
				t.Fatal(e)
			}
			if tc.code == 403 {
				var apiErr *ErrorResponse
				if !errors.As(e, &apiErr) || resp == nil || resp.StatusCode != 403 {
					t.Fatal(resp, e)
				}
			}
			expectedCloses := 0
			if tc.code != 0 {
				expectedCloses = 1
			}
			if f.closed != expectedCloses {
				t.Fatal("close", f.closed)
			}
		})
	}
}
func TestSnapshotUpdateLocalFailuresPreserveInput(t *testing.T) {
	f := &snapshotUpdateWire{t: t}
	c := snapshotUpdateClient(f)
	input := &SnapshotUpdateRequest{Snapshot: &Snapshot{UUID: "caller-id", Name: "renamed", Meta: map[string]interface{}{"bad": make(chan int)}}}
	for _, id := range []string{"", "%", "target-id"} {
		_, _, e := c.Snapshots.Update(context.Background(), id, input)
		if e == nil || input.UUID != "caller-id" || input.Tags != nil || f.calls != 0 {
			t.Fatal(id, e, input, f.calls)
		}
	}
}
func TestSnapshotUpdateNilEmbeddedPayload(t *testing.T) {
	f := &snapshotUpdateWire{t: t}
	c := snapshotUpdateClient(f)
	for _, input := range []*SnapshotUpdateRequest{nil, {}} {
		got, resp, e := c.Snapshots.Update(context.Background(), "target-id", input)
		if !errors.Is(e, ErrEmptyPayloadNotAllowed) || got != nil || resp != nil || f.calls != 0 {
			t.Fatal(got, resp, e, f.calls)
		}
	}
}
