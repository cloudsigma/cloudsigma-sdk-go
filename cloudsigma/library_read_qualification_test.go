package cloudsigma

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

// Sanitized public-shaped fixture, NOT literal complete public-response equality.
func TestLibraryDriveProjectionCharacterization(t *testing.T) {
	b, e := os.ReadFile("testdata/library_drive_sanitized.json")
	if e != nil {
		t.Fatal(e)
	}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
		if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/libdrives/c7105ec8-5d60-43a7-a11b-18dcfc1a4433/" {
			t.Fatal("wire")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(b))), Request: q}, nil
	})}))
	v, _, e := c.LibraryDrives.Get(context.Background(), "c7105ec8-5d60-43a7-a11b-18dcfc1a4433")
	if e != nil {
		t.Fatal(e)
	}
	raw, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	var fields map[string]json.RawMessage
	if e = json.Unmarshal(raw, &fields); e != nil {
		t.Fatal(e)
	}
	for _, k := range []string{"affinities", "jobs", "owner"} {
		if _, ok := fields[k]; ok {
			t.Errorf("unexpected change to characterized projection: %s", k)
		}
	}
}
