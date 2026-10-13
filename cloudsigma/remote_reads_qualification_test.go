package cloudsigma

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// Synthetic characterization only: public examples conflict with remote headings.
func TestRemoteReadSyntheticWireOwnership(t *testing.T) {
	for _, list := range []bool{false, true} {
		calls := 0
		body := `{"uuid":"synthetic-id","location":"SYN","meta":{"nested":{"value":"original"}},"drive_meta":{"name":"synthetic","size":42},"allocated_size":null,"future":true}`
		uri := "/api/2.0/remotesnapshots/synthetic-id/"
		if list {
			body = `{"objects":[` + body + `],"meta":{"total_count":1}}`
			uri = "/api/2.0/remotesnapshots/detail/"
		}
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			calls++
			if q.Method != "GET" || q.URL.RequestURI() != uri || (q.Body != nil && q.Body != http.NoBody) {
				t.Fatal("wire")
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}))
		read := func() *RemoteSnapshot {
			if list {
				v, r, e := c.RemoteSnapshots.List(context.Background(), nil)
				if e != nil || len(v) != 1 || r.Meta.TotalCount != 1 {
					t.Fatal("list", e)
				}
				return &v[0]
			}
			v, r, e := c.RemoteSnapshots.Get(context.Background(), "synthetic-id")
			if e != nil || r.StatusCode != 200 {
				t.Fatal("get", e)
			}
			return v
		}
		a := read()
		if calls != 1 || a.UUID != "synthetic-id" || a.Location != "SYN" || a.RemoteSnapshotDriveMetadata.Size != 42 || a.AllocatedSize != 0 {
			t.Fatal("synthetic projection")
		}
		expected := read()
		a.Meta["nested"].(map[string]interface{})["value"] = "changed"
		a.RemoteSnapshotDriveMetadata.Name = "changed"
		fresh := read()
		if calls != 3 || !reflect.DeepEqual(fresh, expected) {
			t.Fatal("shared response data")
		}
	}
}

func TestRemoteReadRepresentativeFailures(t *testing.T) {
	for _, list := range []bool{false, true} {
		sentinel := errors.New("synthetic transport")
		for _, code := range []int{403, 404, 500, 0} {
			calls := 0
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
				calls++
				if code == 0 {
					return nil, sentinel
				}
				return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[]`)), Request: q}, nil
			})}))
			var r *Response
			var e error
			if list {
				v, resp, err := c.RemoteSnapshots.List(context.Background(), nil)
				if v != nil {
					t.Fatal("value")
				}
				r, e = resp, err
			} else {
				v, resp, err := c.RemoteSnapshots.Get(context.Background(), "synthetic-id")
				if v != nil {
					t.Fatal("value")
				}
				r, e = resp, err
			}
			if e == nil || calls != 1 || (code != 0 && (r == nil || r.StatusCode != code)) || (code == 0 && !errors.Is(e, sentinel)) {
				t.Fatal("propagation")
			}
		}
		for _, body := range []string{`{`, `[]`} {
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
			})}))
			var e error
			if list {
				_, _, e = c.RemoteSnapshots.List(context.Background(), nil)
			} else {
				_, _, e = c.RemoteSnapshots.Get(context.Background(), "synthetic-id")
			}
			if e == nil {
				t.Fatal("malformed")
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) { return nil, q.Context().Err() })}))
		var e error
		if list {
			_, _, e = c.RemoteSnapshots.List(ctx, nil)
		} else {
			_, _, e = c.RemoteSnapshots.Get(ctx, "synthetic-id")
		}
		if !errors.Is(e, context.Canceled) {
			t.Fatal("cancel", e)
		}
	}
}

func TestRemoteGetEmptyIDNoTransport(t *testing.T) {
	calls := 0
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected") })}))
	v, r, e := c.RemoteSnapshots.Get(context.Background(), "")
	if v != nil || r != nil || e != ErrEmptyArgument || calls != 0 {
		t.Fatal("empty id")
	}
}
