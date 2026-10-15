package cloudsigma

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
)

// Identical full-response oracle; only the selected public read differs.
func TestDriveReadProjection(t *testing.T) {
	body, err := os.ReadFile("testdata/drive_singleton_public.json")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
		calls++
		if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/drives/5d3bb6d8-ce39-41e8-8742-2ba5af5ef2d0/" || q.Body != nil && q.Body != http.NoBody {
			t.Fatal("wire", q)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: q}, nil
	})}))
	value, r, err := c.Drives.GetRaw(context.Background(), "5d3bb6d8-ce39-41e8-8742-2ba5af5ef2d0")
	if err != nil || r == nil || calls != 1 {
		t.Fatal(err, calls)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var got, want interface{}
	decode := func(b []byte, v *interface{}) {
		d := json.NewDecoder(strings.NewReader(string(b)))
		d.UseNumber()
		if e := d.Decode(v); e != nil {
			t.Fatal(e)
		}
	}
	decode(encoded, &got)
	decode(body, &want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("documented drive projection lost fields: got %s; want %s", encoded, body)
	}
}
