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

// Complete retained public example; metadata's reported count is not recomputed.
func TestLocationsCompletePublicQualification(t *testing.T) {
	b, e := os.ReadFile("testdata/locations_public.json")
	if e != nil {
		t.Fatal(e)
	}
	var expected struct {
		Objects []map[string]interface{} `json:"objects"`
		Meta    map[string]interface{}   `json:"meta"`
	}
	if e = json.Unmarshal(b, &expected); e != nil {
		t.Fatal(e)
	}
	calls := 0
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithLocation("next"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
		calls++
		if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/locations/" || (q.Body != nil && q.Body != http.NoBody) {
			t.Fatal("wire")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(b))), Request: q}, nil
	})}))
	read := func() []Location {
		v, r, e := c.Locations.List(context.Background())
		if e != nil || r.StatusCode != 200 || r.Meta == nil || r.Meta.TotalCount != 16 {
			t.Fatal("read", e)
		}
		raw, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		var got []map[string]interface{}
		if e = json.Unmarshal(raw, &got); e != nil {
			t.Fatal(e)
		}
		if !reflect.DeepEqual(got, expected.Objects) {
			t.Fatal("public fields")
		}
		return v
	}
	a := read()
	if calls != 1 || len(a) != 17 {
		t.Fatal("count")
	}
	a[0].DisplayName = "changed"
	fresh := read()
	if calls != 2 || fresh[0].DisplayName == "changed" {
		t.Fatal("ownership")
	}
}
