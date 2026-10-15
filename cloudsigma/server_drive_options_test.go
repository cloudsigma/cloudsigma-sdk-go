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

type attachmentWire struct {
	t       *testing.T
	want    string
	code    int
	body    string
	calls   int
	failure error
}

func (f *attachmentWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "PUT" || q.URL.RequestURI() != "/api/2.0/servers/server/" {
		f.t.Fatalf("route %s %s", q.Method, q.URL)
	}
	b, e := io.ReadAll(q.Body)
	if e != nil {
		f.t.Fatal(e)
	}
	var got, want map[string]json.RawMessage
	if json.Unmarshal(b, &got) != nil || json.Unmarshal([]byte(f.want), &want) != nil || !reflect.DeepEqual(got, want) {
		f.t.Fatalf("body %s want %s", b, f.want)
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	if f.failure != nil {
		return nil, f.failure
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.body)), Request: q}, nil
}
func TestDriveOptionsPresence(t *testing.T) {
	zero, big := int64(0), int64(9007199254740993)
	for _, tc := range []struct {
		name    string
		options []ServerDriveOptions
		want    string
	}{
		{"empty", []ServerDriveOptions{}, `{"drives":[]}`},
		{"omitted", []ServerDriveOptions{{nil, "0:0", "virtio", "drive"}}, `{"drives":[{"dev_channel":"0:0","device":"virtio","drive":"drive"}]}`},
		{"zero", []ServerDriveOptions{{&zero, "0:0", "virtio", "drive"}}, `{"drives":[{"boot_order":0,"dev_channel":"0:0","device":"virtio","drive":"drive"}]}`},
		{"ordered_numeric", []ServerDriveOptions{{&big, "0:1", "virtio", "second"}, {&zero, "0:0", "ide", "first"}}, `{"drives":[{"boot_order":9007199254740993,"dev_channel":"0:1","device":"virtio","drive":"second"},{"boot_order":0,"dev_channel":"0:0","device":"ide","drive":"first"}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &attachmentWire{t: t, want: tc.want, code: 200, body: `{"uuid":"server","status":"stopped","drives":[{"drive":{"uuid":"result"}}],"jobs":[]}`}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			before, _ := json.Marshal(tc.options)
			out, r, e := c.Servers.UpdateDriveOptions(context.Background(), "server", tc.options)
			after, _ := json.Marshal(tc.options)
			if e != nil || r == nil || out == nil || out.UUID != "server" || string(out.Jobs) != "[]" || f.calls != 1 || string(before) != string(after) {
				t.Fatalf("result %v %v %v", out, r, e)
			}
			out.Drives[0].Drive.UUID = "mutated"
			if out.Drives[0].Drive.UUID != "mutated" {
				t.Fatal("mutation not applied")
			}
			after, _ = json.Marshal(tc.options)
			if string(after) != string(before) {
				t.Fatal("result aliases input")
			}
		})
	}
}
func TestDriveOptionsFailures(t *testing.T) {
	f := &attachmentWire{t: t, want: `{"drives":[]}`, code: 403, body: `[{"error_message":"denied","error_point":"server","error_type":"permission"}]`}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	s, r, e := c.Servers.UpdateDriveOptions(context.Background(), "server", []ServerDriveOptions{})
	var api *ErrorResponse
	if s != nil || r == nil || r.StatusCode != 403 || !errors.As(e, &api) || api.Response != r || !reflect.DeepEqual(api.Errors, []Error{{Message: "denied", Point: "server", Type: "permission"}}) || f.calls != 1 {
		t.Fatal("HTTP error fidelity")
	}
	for _, body := range []string{"", `{`, `[]`, `{"drives":"wrong"}`} {
		f.code = 200
		f.body = body
		f.calls = 0
		s, _, e = c.Servers.UpdateDriveOptions(context.Background(), "server", []ServerDriveOptions{})
		if e == nil || s != nil || f.calls != 1 {
			t.Fatalf("malformed %q", body)
		}
	}
	f.calls = 0
	for _, id := range []string{"", "../escape", "server?x=1"} {
		_, _, e = c.Servers.UpdateDriveOptions(context.Background(), id, []ServerDriveOptions{})
		if e == nil {
			t.Fatal("identifier")
		}
	}
	_, _, e = c.Servers.UpdateDriveOptions(context.Background(), "server", nil)
	if !errors.Is(e, ErrEmptyPayloadNotAllowed) || f.calls != 0 {
		t.Fatal("nil/zero dispatch")
	}
	sentinel := errors.New("synthetic transport")
	f.failure = sentinel
	_, _, e = c.Servers.UpdateDriveOptions(context.Background(), "server", []ServerDriveOptions{})
	if !errors.Is(e, sentinel) {
		t.Fatal("transport")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, e = c.Servers.UpdateDriveOptions(ctx, "server", []ServerDriveOptions{})
	if !errors.Is(e, context.Canceled) {
		t.Fatal("cancellation")
	}
	// 202 is not the documented edit success status. Characterize the common
	// transport's acceptance without claiming an applied runtime change.
	f.failure = nil
	f.code = 202
	f.body = `{"uuid":"server","status":"starting"}`
	s, r, e = c.Servers.UpdateDriveOptions(context.Background(), "server", []ServerDriveOptions{})
	if e != nil || r.StatusCode != 202 || s.Status != "starting" {
		t.Fatal("202 acknowledgment characterization")
	}
}

func TestDriveOptionsInputSnapshot(t *testing.T) {
	order := int64(0)
	options := []ServerDriveOptions{{&order, "0:0", "virtio", "original"}}
	w := &attachmentMutationWire{t: t, options: options, order: &order}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: w}))
	_, _, err := c.Servers.UpdateDriveOptions(context.Background(), "server", options)
	if err != nil || !w.mutated || order != 9 || options[0].Drive != "changed" {
		t.Fatal("mutation did not occur", err)
	}
}

type attachmentMutationWire struct {
	t       *testing.T
	options []ServerDriveOptions
	order   *int64
	mutated bool
}

func (w *attachmentMutationWire) RoundTrip(r *http.Request) (*http.Response, error) {
	// Synchronous encoding has completed before dispatch. This is a snapshot
	// test, not a promise of concurrency safety or retained deep-copy ownership.
	*w.order = 9
	w.options[0].Drive = "changed"
	w.mutated = true
	b, e := io.ReadAll(r.Body)
	if e != nil {
		w.t.Fatal(e)
	}
	if strings.TrimSpace(string(b)) != `{"drives":[{"boot_order":0,"dev_channel":"0:0","device":"virtio","drive":"original"}]}` {
		w.t.Fatalf("snapshot %s", b)
	}
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"server"}`)), Request: r}, nil
}
