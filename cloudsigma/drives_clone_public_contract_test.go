package cloudsigma

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// Public contract: https://cloudsigma-docs.readthedocs.io/en/latest/drives.html#cloning
// Optional body and partial drive definition; 202 is acceptance, not completion.
type publicCloneWire struct {
	t     *testing.T
	want  string
	calls int
}

func (f *publicCloneWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != http.MethodPost || q.URL.RequestURI() != "/api/2.0/drives/public-source/action/?do=clone" {
		f.t.Fatalf("unexpected request %s %s", q.Method, q.URL.RequestURI())
	}
	body := ""
	if q.Body != nil {
		b, e := io.ReadAll(q.Body)
		if e != nil {
			f.t.Fatal(e)
		}
		body = strings.TrimSpace(string(b))
	}
	if body != f.want {
		f.t.Fatalf("wire body got %q want %q", body, f.want)
	}
	return &http.Response{StatusCode: http.StatusAccepted, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[{"uuid":"accepted-clone"}]}`)), Request: q}, nil
}
func TestDrivesClonePublicOptionalBodyAccepted(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request *DriveCloneRequest
		wire    string
	}{
		{"omitted", nil, ""},
		{"name_only", &DriveCloneRequest{Drive: &Drive{Name: "public-clone"}}, `{"name":"public-clone"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &publicCloneWire{t: t, want: tc.wire}
			c := NewClient(NewTokenCredentialsProvider("synthetic-only"), WithHTTPClient(&http.Client{Transport: f}))
			d, r, e := c.Drives.Clone(context.Background(), "public-source", tc.request)
			if e != nil || d == nil || d.UUID != "accepted-clone" || r == nil || r.StatusCode != http.StatusAccepted || f.calls != 1 {
				t.Fatalf("acceptance result drive=%v response=%v err=%v calls=%d", d, r, e, f.calls)
			}
		})
	}
}
