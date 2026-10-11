package cloudsigma

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type schedulerWire struct {
	t     *testing.T
	calls int
	err   error
	code  int
	body  *string
}

func (f *schedulerWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if err := q.Context().Err(); err != nil {
		return nil, err
	}
	if f.err != nil {
		return nil, f.err
	}
	if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/drives/34f9c721-ec23-4f5c-86ab-2839f687f14e/action/?do=set_scheduler" {
		f.t.Fatal("route", q.URL)
	}
	var body map[string]string
	if err := json.NewDecoder(q.Body).Decode(&body); err != nil {
		f.t.Fatal(err)
	}
	if len(body) != 1 || body["backup_policy_uuid"] != "103de42c-17a1-40ea-bc0f-521789168bbf" {
		f.t.Fatal(body)
	}
	if f.body != nil {
		return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(*f.body)), Request: q}, nil
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"34f9c721-ec23-4f5c-86ab-2839f687f14e","backup_scheduler":{"uuid":"103de42c-17a1-40ea-bc0f-521789168bbf","resource_uri":"/api/2.0/backupschedulers/103de42c-17a1-40ea-bc0f-521789168bbf/"},"licenses":[{"license":{"free":0}}]}`)), Request: q}, nil
}
func TestSetSchedulerPublicBinding(t *testing.T) {
	f := &schedulerWire{t: t, code: 202}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	d, r, e := c.Drives.SetScheduler(context.Background(), "34f9c721-ec23-4f5c-86ab-2839f687f14e", "103de42c-17a1-40ea-bc0f-521789168bbf")
	if e != nil || r == nil || r.StatusCode != 202 || d == nil || d.BackupScheduler == nil || d.BackupScheduler.UUID != "103de42c-17a1-40ea-bc0f-521789168bbf" || d.BackupScheduler.ResourceURI != "/api/2.0/backupschedulers/103de42c-17a1-40ea-bc0f-521789168bbf/" || d.UUID != "34f9c721-ec23-4f5c-86ab-2839f687f14e" || len(d.Licenses) != 1 || d.Licenses[0].License.Free != "0" || f.calls != 1 {
		t.Fatalf("binding %v %v %v", d, r, e)
	}
}
func TestSetSchedulerRefusalAndErrors(t *testing.T) {
	f := &schedulerWire{t: t, code: 202}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	for _, id := range []string{"", "..", "../../servers/id", "id?do=delete", "%2f", "id\\x", "id\n"} {
		for _, ids := range [][2]string{{id, "valid"}, {"valid", id}} {
			if _, _, e := c.Drives.SetScheduler(context.Background(), ids[0], ids[1]); e == nil {
				t.Fatal("accepted unsafe")
			}
		}
	}
	if f.calls != 0 {
		t.Fatal("dispatched invalid")
	}
	a, b := "34f9c721-ec23-4f5c-86ab-2839f687f14e", "103de42c-17a1-40ea-bc0f-521789168bbf"
	sentinel := errors.New("synthetic")
	f.err = sentinel
	if _, _, e := c.Drives.SetScheduler(context.Background(), a, b); !errors.Is(e, sentinel) {
		t.Fatal(e)
	}
	f.err = nil
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, e := c.Drives.SetScheduler(ctx, a, b); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	f.code = 403
	if d, r, e := c.Drives.SetScheduler(context.Background(), a, b); e == nil || d != nil || r == nil || r.StatusCode != 403 {
		t.Fatal("error lost")
	}
}

// Characterizes permissive existing decoding; not a normative schema assertion.
func TestSetSchedulerLinkDecodeBoundaries(t *testing.T) {
	for _, tc := range []struct {
		body    string
		fail    bool
		nilLink bool
	}{
		{`{"uuid":"id"}`, false, true}, {`{"uuid":"id","backup_scheduler":null}`, false, true}, {`null`, false, true}, {`{"backup_scheduler":{}}`, false, false}, {`{"backup_scheduler":"bad"}`, true, true}, {`{`, true, true}, {`[]`, true, true},
	} {
		t.Run(tc.body, func(t *testing.T) {
			f := &schedulerWire{t: t, code: 202, body: &tc.body}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			out, _, e := c.Drives.SetScheduler(context.Background(), "34f9c721-ec23-4f5c-86ab-2839f687f14e", "103de42c-17a1-40ea-bc0f-521789168bbf")
			if (e != nil) != tc.fail {
				t.Fatalf("decode %v", e)
			}
			if !tc.fail && (out == nil || (out.BackupScheduler == nil) != tc.nilLink) {
				t.Fatal("link shape changed")
			}
		})
	}
}
