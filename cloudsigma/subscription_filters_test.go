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

func TestSubscriptionFilterWire(t *testing.T) {
	for _, status := range []string{"", "active", "inactive", "expired", "all", "notexpired"} {
		opts := &SubscriptionListOptions{Status: status, Resources: []string{"cpu", "mem"}}
		before := append([]string(nil), opts.Resources...)
		calls := 0
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			calls++
			want := "/api/2.0/subscriptions/?resource=cpu%2Cmem"
			if status != "" {
				want += "&status=" + status
			}
			if q.Method != "GET" || q.URL.RequestURI() != want || q.URL.Query().Get("resource") != "cpu,mem" || (q.Body != nil && q.Body != http.NoBody) {
				t.Fatal("wire")
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[{"uuid":"synthetic","status":"active","amount":"1.00000000000001"}],"meta":{"total_count":1}}`)), Request: q}, nil
		})}))
		v, r, e := c.Subscriptions.ListWithOptions(context.Background(), opts)
		if e != nil || calls != 1 || len(v) != 1 || v[0].Status != "active" || v[0].Amount != "1.00000000000001" || r.Meta.TotalCount != 1 {
			t.Fatal("read", e)
		}
		if !reflect.DeepEqual(opts.Resources, before) || opts.Status != status {
			t.Fatal("mutated options")
		}
		v[0].Status = "changed"
		again, _, e := c.Subscriptions.ListWithOptions(context.Background(), opts)
		if e != nil || again[0].Status != "active" || calls != 2 {
			t.Fatal("refetch")
		}
	}
}
func TestSubscriptionFilterOmissionAndInvalid(t *testing.T) {
	calls := 0
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
		calls++
		if q.URL.RequestURI() != "/api/2.0/subscriptions/" {
			t.Fatal("omission")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[]}`)), Request: q}, nil
	})}))
	for _, o := range []*SubscriptionListOptions{nil, {}, {Resources: []string{}}} {
		v, _, e := c.Subscriptions.ListWithOptions(context.Background(), o)
		if e != nil || len(v) != 0 {
			t.Fatal(e)
		}
	}
	if _, _, e := c.Subscriptions.List(context.Background()); e != nil || calls != 4 {
		t.Fatal("legacy")
	}
	for _, o := range []*SubscriptionListOptions{{Status: "active&x=1"}, {Status: "ACTIVE"}, {Resources: []string{""}}, {Resources: []string{"cpu,mem"}}, {Resources: []string{"cpu", "unknown"}}} {
		v, r, e := c.Subscriptions.ListWithOptions(context.Background(), o)
		if v != nil || r != nil || e == nil || calls != 4 {
			t.Fatal("invalid dispatched")
		}
	}
}
func TestSubscriptionFilterErrors(t *testing.T) {
	sentinel := errors.New("synthetic")
	for _, code := range []int{403, 404, 500, 0} {
		calls := 0
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			calls++
			if code == 0 {
				return nil, sentinel
			}
			return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[]`)), Request: q}, nil
		})}))
		v, r, e := c.Subscriptions.ListWithOptions(context.Background(), nil)
		if v != nil || e == nil || calls != 1 || (code != 0 && (r == nil || r.StatusCode != code)) || (code == 0 && !errors.Is(e, sentinel)) {
			t.Fatal("error")
		}
	}
	for _, body := range []string{`{`, `[]`} {
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}))
		if _, _, e := c.Subscriptions.ListWithOptions(context.Background(), nil); e == nil {
			t.Fatal("decode")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) { return nil, q.Context().Err() })}))
	if _, _, e := c.Subscriptions.ListWithOptions(ctx, nil); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}

// Existing typed projection intentionally remains unchanged, not full-field raw fidelity.
func TestSubscriptionFilterPublicProjection(t *testing.T) {
	b, e := os.ReadFile("testdata/subscriptions_public.json")
	if e != nil {
		t.Fatal(e)
	}
	var root subscriptionsRoot
	if e = json.Unmarshal(b, &root); e != nil {
		t.Fatal(e)
	}
	calls := 0
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
		calls++
		if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/subscriptions/?resource=dssd%2Ccpu%2Cmem%2Ctx%2Cip%2Cvlan&status=all" || (q.Body != nil && q.Body != http.NoBody) {
			t.Fatal("wire")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(b))), Request: q}, nil
	})}))
	opts := &SubscriptionListOptions{Status: "all", Resources: []string{"dssd", "cpu", "mem", "tx", "ip", "vlan"}}
	v, r, e := c.Subscriptions.ListWithOptions(context.Background(), opts)
	if e != nil || calls != 1 || !reflect.DeepEqual(v, root.Subscriptions) || !reflect.DeepEqual(r.Meta, root.Meta) {
		t.Fatal("projection", e)
	}
	opts.Resources[0] = "cpu"
	if v[0].Resource != "dssd" {
		t.Fatal("options alias response")
	}
}
