package cloudsigma

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestDiscountListRaw(t *testing.T) {
	// First fixture follows the public billing.html discount response; the second
	// tests synthetic extensions only, not additional provider wire guarantees.
	fixtures := []string{
		`{"meta":{"limit":0,"offset":0,"total_count":5},"objects":[{"period":"3 months","value":"0.0300000000"},{"period":"6 months","value":"0.1000000000"},{"period":"1 year","value":"0.2500000000"},{"period":"2 years","value":"0.3500000000"},{"period":"3 years","value":"0.4500000000"}]}`,
		`{"meta":{"limit":2,"offset":3,"total_count":9,"future":9007199254740993},"objects":[{"period":null,"value":"0.0300000000","future":[3,2,1]}],"extension":9007199254740993.0000000001}`,
		`{}`, `{"objects":null}`, `{"objects":[]}`, `{"meta":null}`,
	}
	for _, tc := range []struct {
		opts *ListOptions
		uri  string
	}{
		{nil, "/api/2.0/discount/"},
		{&ListOptions{}, "/api/2.0/discount/?limit=0"},
		{&ListOptions{Limit: 2, Offset: 3}, "/api/2.0/discount/?limit=2&offset=3"},
		{&ListOptions{Limit: 2}, "/api/2.0/discount/?limit=2"},
	} {
		for _, body := range fixtures {
			before, _ := json.Marshal(tc.opts)
			calls := 0
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
				calls++
				if q.Method != "GET" || q.URL.RequestURI() != tc.uri {
					t.Fatalf("wire: %s %s", q.Method, q.URL.RequestURI())
				}
				if q.Body != nil && q.Body != http.NoBody {
					b, e := io.ReadAll(q.Body)
					if e != nil || len(b) != 0 {
						t.Fatal("body", e)
					}
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
			})}))
			a, r, e := c.Discount.ListRaw(context.Background(), tc.opts)
			if e != nil || a == nil || r == nil || r.StatusCode != 200 || calls != 1 {
				t.Fatal("list", e, calls)
			}
			var expected map[string]json.RawMessage
			if err := json.Unmarshal([]byte(body), &expected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(map[string]json.RawMessage(*a), expected) {
				t.Fatal("envelope/precision/null/order")
			}
			after, _ := json.Marshal(tc.opts)
			if string(before) != string(after) {
				t.Fatal("options mutated")
			}
			(*a)["injected"] = json.RawMessage(`true`)
			if raw := (*a)["objects"]; len(raw) > 0 {
				raw[0] = 'X'
				if (*a)["objects"][0] != 'X' {
					t.Fatal("mutation failed")
				}
			}
			b, _, e := c.Discount.ListRaw(context.Background(), tc.opts)
			if e != nil || calls != 2 || !reflect.DeepEqual(map[string]json.RawMessage(*b), expected) {
				t.Fatal("ownership", e)
			}
		}
	}
}

func TestDiscountErrors(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `1`, `"x"`, `{`, ``} {
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}))
		a, r, e := c.Discount.ListRaw(context.Background(), nil)
		if a != nil || r != nil || e == nil {
			t.Fatal("malformed convention", body, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	for _, code := range []int{403, 404, 500, 0} {
		calls := 0
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			calls++
			if code == 0 {
				return nil, sentinel
			}
			return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[{"error_type":"permission","error_message":"synthetic denied","error_point":null}]`)), Request: q}, nil
		})}))
		a, r, e := c.Discount.ListRaw(context.Background(), nil)
		if a != nil || e == nil || calls != 1 {
			t.Fatal("error", e)
		}
		if code == 0 {
			if !errors.Is(e, sentinel) || r != nil {
				t.Fatal("transport", e)
			}
		} else {
			var api *ErrorResponse
			if !errors.As(e, &api) || r == nil || r.StatusCode != code || len(api.Errors) != 1 || api.Errors[0].Message != "synthetic denied" || api.Errors[0].Type != "permission" || api.Errors[0].Point != "" || api.Response != r {
				t.Fatal("typed error", e)
			}
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) { return nil, q.Context().Err() })}))
	if _, _, e := c.Discount.ListRaw(ctx, nil); !errors.Is(e, context.Canceled) {
		t.Fatal("cancel", e)
	}
}
