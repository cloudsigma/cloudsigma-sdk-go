package cloudsigma

import (
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

func TestSnapshotsDetailRawComplete(t *testing.T) {
	pub, err := os.ReadFile("testdata/snapshots_detail_public.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{string(pub), `{"amount":9007199254740993.00000000000001,"synthetic_currency":"XXX","future":{"n":12345678901234567890},"optional":null}`, `{}`, `{"amount":null}`, `{"objects":[]}`, `{"objects":null}`, `{"amount":"-0.0000000000000000001","optional":"1e99","synthetic_currency":"synthetic"}`} {
		calls := 0
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithLocation("next"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			calls++
			if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/snapshots/detail/" || q.Body != nil && q.Body != http.NoBody {
				t.Fatal("wire")
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}))
		a, r, e := c.Snapshots.ListDetailRaw(context.Background())
		if e != nil || r.StatusCode != 200 || calls != 1 {
			t.Fatal("get", e)
		}
		if strings.Contains(body, "9007199254740993") {
			if string((*a)["amount"]) != "9007199254740993.00000000000001" || string((*a)["future"]) != `{"n":12345678901234567890}` {
				t.Fatal("precision")
			}
		}
		if body == string(pub) {
			var expected map[string]json.RawMessage
			if err := json.Unmarshal(pub, &expected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(map[string]json.RawMessage(*a), expected) {
				t.Fatal("complete public fixture")
			}
		}

		if body == `{}` {
			if _, ok := (*a)["amount"]; ok {
				t.Fatal("missing defaulted")
			}
		}
		// Mutate both map membership and owned bytes; a fresh response must be independent.
		(*a)["injected"] = []byte(`true`)
		if raw := (*a)["objects"]; len(raw) > 0 {
			raw[0] = 'X'
		}
		if v := (*a)["amount"]; len(v) > 0 {
			v[0] = 'X'
		}
		b, _, e := c.Snapshots.ListDetailRaw(context.Background())
		if e != nil || calls != 2 {
			t.Fatal(e)
		}
		var expected map[string]json.RawMessage
		if err := json.Unmarshal([]byte(body), &expected); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(map[string]json.RawMessage(*b), expected) {
			t.Fatal("refetch fidelity after mutation")
		}
		if _, ok := (*b)["injected"]; ok {
			t.Fatal("shared map")
		}
		if v := (*b)["amount"]; len(v) > 0 && v[0] == 'X' {
			t.Fatal("shared bytes")
		}
	}
}
func TestSnapshotsDetailRawErrors(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `1`, `"x"`, `{`, ``} {
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}))
		if _, _, e := c.Snapshots.ListDetailRaw(context.Background()); e == nil {
			t.Fatal("shape", body)
		}
	}
	sentinel := errors.New("transport")
	for _, code := range []int{403, 404, 500, 0} {
		calls := 0
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			calls++
			if code == 0 {
				return nil, sentinel
			}
			return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[]`)), Request: q}, nil
		})}))
		v, r, e := c.Snapshots.ListDetailRaw(context.Background())
		if v != nil || e == nil || calls != 1 || code != 0 && (r == nil || r.StatusCode != code) || code == 0 && !errors.Is(e, sentinel) {
			t.Fatal("propagation")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) { return nil, q.Context().Err() })}))
	if _, _, e := c.Snapshots.ListDetailRaw(ctx); !errors.Is(e, context.Canceled) {
		t.Fatal("cancel", e)
	}
}
