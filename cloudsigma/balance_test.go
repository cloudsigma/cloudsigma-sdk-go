package cloudsigma

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestBalanceGetRaw(t *testing.T) {
	pub, err := os.ReadFile("testdata/balance_public.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{string(pub), `{"balance":9007199254740993.00000000000001,"currency":"XXX","future":{"n":12345678901234567890},"credit_limit":null}`, `{}`, `{"balance":null}`, `{"balance":"-0.0000000000000000001","credit_limit":"1e99","currency":"synthetic"}`} {
		calls := 0
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithLocation("next"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			calls++
			if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/balance/" || q.Body != nil && q.Body != http.NoBody {
				t.Fatal("wire")
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}))
		a, r, e := c.Balance.Get(context.Background())
		if e != nil || r.StatusCode != 200 || calls != 1 {
			t.Fatal("get", e)
		}
		if strings.Contains(body, "9007199254740993") {
			if string((*a)["balance"]) != "9007199254740993.00000000000001" || string((*a)["future"]) != `{"n":12345678901234567890}` {
				t.Fatal("precision")
			}
		}
		if body == string(pub) {
			if string((*a)["balance"]) != `"155367.99237092264288897986"` || string((*a)["currency"]) != `"EUR"` || string((*a)["credit_limit"]) != "null" {
				t.Fatal("public fixture")
			}
		}
		if body == `{}` {
			if _, ok := (*a)["balance"]; ok {
				t.Fatal("missing defaulted")
			}
		}
		// Mutate both map membership and owned bytes; a fresh response must be independent.
		(*a)["injected"] = []byte(`true`)
		if v := (*a)["balance"]; len(v) > 0 {
			v[0] = 'X'
		}
		b, _, e := c.Balance.Get(context.Background())
		if e != nil || calls != 2 {
			t.Fatal(e)
		}
		if _, ok := (*b)["injected"]; ok {
			t.Fatal("shared map")
		}
		if v := (*b)["balance"]; len(v) > 0 && v[0] == 'X' {
			t.Fatal("shared bytes")
		}
	}
}
func TestBalanceErrors(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `1`, `"x"`, `{`, ``} {
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}))
		if _, _, e := c.Balance.Get(context.Background()); e == nil {
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
		v, r, e := c.Balance.Get(context.Background())
		if v != nil || e == nil || calls != 1 || code != 0 && (r == nil || r.StatusCode != code) || code == 0 && !errors.Is(e, sentinel) {
			t.Fatal("propagation")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) { return nil, q.Context().Err() })}))
	if _, _, e := c.Balance.Get(ctx); !errors.Is(e, context.Canceled) {
		t.Fatal("cancel", e)
	}
}
