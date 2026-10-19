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

// The clone contract accepts optional drive-definition fields; allow_multimount
// is a writable boolean. False must be distinguishable from no override.
type cloneMultimountWire struct {
	t        *testing.T
	want     string
	code     int
	response string
	calls    int
	failure  error
}

func (f *cloneMultimountWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if err := q.Context().Err(); err != nil {
		return nil, err
	}
	if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/drives/source/action/?do=clone" {
		f.t.Fatalf("route %s %s", q.Method, q.URL.RequestURI())
	}
	var b []byte
	if q.Body != nil {
		var e error
		b, e = io.ReadAll(q.Body)
		if e != nil {
			f.t.Fatal(e)
		}
	}
	if f.want == "" {
		if len(b) != 0 {
			f.t.Fatalf("unexpected body %s", b)
		}
	} else {
		var got, want map[string]json.RawMessage
		if json.Unmarshal(b, &got) != nil || json.Unmarshal([]byte(f.want), &want) != nil || !reflect.DeepEqual(got, want) {
			f.t.Fatalf("strict body got %s want %s", b, f.want)
		}
	}
	if f.failure != nil {
		return nil, f.failure
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.response)), Request: q}, nil
}
func TestCloneMultimountPresence(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  string
	}{
		{"absent", "{}", "{}"},
		{"false", `{"allow_multimount":false}`, `{"allow_multimount":false}`},
		{"true", `{"allow_multimount":true}`, `{"allow_multimount":true}`},
		{"false_with_definition", `{"name":"clone","media":"disk","size":9007199254740993,"allow_multimount":false}`, `{"name":"clone","media":"disk","size":9007199254740993,"allow_multimount":false}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Decode the independently specified fixture, not the production options encoder.
			var input struct {
				*Drive
				AllowMultimount *bool `json:"allow_multimount"`
			}
			if e := json.Unmarshal([]byte(tc.input), &input); e != nil {
				t.Fatal(e)
			}
			f := &cloneMultimountWire{t: t, want: tc.want, code: 202, response: `{"objects":[{"uuid":"clone","allow_multimount":false,"status":"cloning_dst","runtime":{"is_snapshotable":true,"snapshots_allocated_size":0,"storage_type":"dssd"}}]}`}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			d, r, e := c.Drives.CloneWithOptions(context.Background(), "source", &DriveCloneOptions{Definition: input.Drive, AllowMultimount: input.AllowMultimount})
			if e != nil || r == nil || r.StatusCode != 202 || d == nil || d.UUID != "clone" || d.AllowMultimount || d.Status != "cloning_dst" || d.Runtime == nil || !d.Runtime.IsSnapshotable || d.Runtime.SnapshotsAllocatedSize != 0 || d.Runtime.StorageType != "dssd" || f.calls != 1 {
				t.Fatalf("acknowledgment: %v %v %v", d, r, e)
			}
		})
	}
}
func TestCloneMultimountFailures(t *testing.T) {
	f := &cloneMultimountWire{t: t, want: "", code: 403, response: `[{"error_message":"denied","error_point":"drive","error_type":"permission"}]`}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	d, r, e := c.Drives.CloneWithOptions(context.Background(), "source", nil)
	var api *ErrorResponse
	if d != nil || r == nil || r.StatusCode != 403 || !errors.As(e, &api) || api.Response != r || !reflect.DeepEqual(api.Errors, []Error{{Message: "denied", Point: "drive", Type: "permission"}}) || f.calls != 1 {
		t.Fatal("HTTP error fidelity")
	}
	for _, body := range []string{"", `{`, `[]`, `{"objects":[{"allow_multimount":"false"}]}`} {
		f.code = 202
		f.response = body
		n := f.calls
		d, r, e = c.Drives.CloneWithOptions(context.Background(), "source", nil)
		if d != nil || r != nil || e == nil || f.calls != n+1 {
			t.Fatal("decode error convention")
		}
	}
	f.response = `{"objects":[]}`
	d, r, e = c.Drives.CloneWithOptions(context.Background(), "source", nil)
	if d != nil || r == nil || r.StatusCode != 202 || e == nil {
		t.Fatal("empty acknowledgment policy")
	}
	f.failure = errors.New("synthetic transport failure")
	d, r, e = c.Drives.CloneWithOptions(context.Background(), "source", nil)
	if d != nil || r != nil || !errors.Is(e, f.failure) {
		t.Fatal("transport error")
	}
	n := f.calls
	_, _, e = c.Drives.CloneWithOptions(context.Background(), "", nil)
	if !errors.Is(e, ErrEmptyArgument) || f.calls != n {
		t.Fatal("empty ID dispatch")
	}
}

func TestCloneMultimountLegacyAndOverride(t *testing.T) {
	no, yes := false, true
	for _, tc := range []struct {
		name    string
		request *DriveCloneOptions
		want    string
	}{
		{"nil_request", nil, ""},
		{"nil_embedded", &DriveCloneOptions{}, "{}"},
		{"legacy_true", &DriveCloneOptions{Definition: &Drive{AllowMultimount: true}}, `{"allow_multimount":true}`},
		{"legacy_false", &DriveCloneOptions{Definition: &Drive{}}, `{}`},
		{"override_false", &DriveCloneOptions{Definition: &Drive{AllowMultimount: true}, AllowMultimount: &no}, `{"allow_multimount":false}`},
		{"override_true", &DriveCloneOptions{Definition: &Drive{}, AllowMultimount: &yes}, `{"allow_multimount":true}`},
		{"override_without_drive", &DriveCloneOptions{AllowMultimount: &no}, `{"allow_multimount":false}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &cloneMultimountWire{t: t, want: tc.want, code: 202, response: `{"objects":[{"uuid":"clone","status":"cloning_dst"}]}`}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			before, marshalErr := json.Marshal(tc.request)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			d, r, e := c.Drives.CloneWithOptions(context.Background(), "source", tc.request)
			after, marshalErr := json.Marshal(tc.request)
			if marshalErr != nil || string(before) != string(after) {
				t.Fatal("options mutated")
			}
			if e != nil || d == nil || d.UUID != "clone" || r == nil || r.StatusCode != 202 || f.calls != 1 {
				t.Fatal("clone acknowledgment")
			}
			if tc.name == "override_false" && !tc.request.Definition.AllowMultimount {
				t.Fatal("input mutated")
			}
		})
	}
}

func TestCloneMultimountCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &cloneMultimountWire{t: t, want: "", failure: errors.New("transport should observe canceled context first")}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	d, r, e := c.Drives.CloneWithOptions(ctx, "source", nil)
	if d != nil || r != nil || !errors.Is(e, context.Canceled) {
		t.Fatalf("cancellation: %v %v %v", d, r, e)
	}
}
