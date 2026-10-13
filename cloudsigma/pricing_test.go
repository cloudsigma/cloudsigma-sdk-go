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

func TestPricingPublicAndPrecision(t *testing.T) {
	pub, e := os.ReadFile("testdata/pricing_public.json")
	if e != nil {
		t.Fatal(e)
	}
	zero, five := 0, 5
	for _, body := range []string{string(pub), `{"objects":[{"price":"0.000000000000000001","n":9007199254740993.000001}],"meta":null,"future":12345678901234567890}`, `{}`, `{"objects":null}`} {
		for _, limit := range []*int{nil, &zero, &five} {
			calls := 0
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithLocation("next"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
				calls++
				want := "/api/2.0/pricing/"
				if limit == &zero {
					want += "?limit=0"
				}
				if limit == &five {
					want += "?limit=5"
				}
				if q.Method != "GET" || q.URL.RequestURI() != want || q.Body != nil && q.Body != http.NoBody {
					t.Fatal("wire")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
			})}))
			v, r, e := c.Pricing.Get(context.Background(), limit)
			if e != nil || r.StatusCode != 200 || calls != 1 {
				t.Fatal(e)
			}
			var want map[string]json.RawMessage
			if e = json.Unmarshal([]byte(body), &want); e != nil {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(map[string]json.RawMessage(*v), want) {
				t.Fatal("raw fidelity")
			}
			(*v)["injected"] = json.RawMessage(`true`)

			if raw, ok := (*v)["objects"]; ok && len(raw) > 0 {
				raw[0] = 'X'
			}

			next, _, e := c.Pricing.Get(context.Background(), limit)
			if e != nil || !reflect.DeepEqual(map[string]json.RawMessage(*next), want) || calls != 2 {
				t.Fatal("ownership")
			}
		}
	}
}
func TestPricingErrors(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `1`, `{`, ``} {
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}))
		if _, _, e := c.Pricing.Get(context.Background(), nil); e == nil {
			t.Fatal("shape")
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
		v, r, e := c.Pricing.Get(context.Background(), nil)
		if v != nil || e == nil || calls != 1 || code != 0 && (r == nil || r.StatusCode != code) || code == 0 && !errors.Is(e, sentinel) {
			t.Fatal("error")
		}
	}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) { return nil, q.Context().Err() })}))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, e := c.Pricing.Get(ctx, nil); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	n := -1
	c = NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(*http.Request) (*http.Response, error) { t.Fatal("negative dispatch"); return nil, nil })}))
	if _, _, e := c.Pricing.Get(context.Background(), &n); e == nil {
		t.Fatal("negative limit")
	}
}
