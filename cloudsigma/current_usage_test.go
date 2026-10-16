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

func TestCurrentUsageGetRaw(t *testing.T) {
	// First fixture is the complete public current_usage.html example. Remaining
	// fixtures characterize synthetic extensions or permissive decoder behavior.
	fixtures := []string{
		`{"balance":{"balance":"155367.99237092264288897986","currency":"EUR"},"usage":{"amd_cpu":{"burst":0,"subscribed":10000,"using":0},"amd_mem":{"burst":0,"subscribed":17179869184,"using":0},"cpu":{"burst":0,"subscribed":10000,"using":0},"dssd":{"burst":0,"subscribed":32212254720,"using":30064771072},"ip":{"burst":0,"subscribed":1,"using":1},"mem":{"burst":0,"subscribed":17179869184,"using":0},"msft_7jq_00341":{"burst":0,"subscribed":0,"using":0},"msft_7nq_00302":{"burst":0,"subscribed":0,"using":0},"msft_p73_04837_core":{"burst":0,"subscribed":0,"using":0},"msft_tfa_00523":{"burst":0,"subscribed":0,"using":0},"sms":{"burst":0,"subscribed":0,"using":0},"sparc_t4_cpu":{"burst":0,"subscribed":0,"using":0},"sparc_t4_mem":{"burst":0,"subscribed":0,"using":0},"ssd":{"burst":0,"subscribed":0,"using":0},"tx":{"burst":0,"subscribed":0,"using":0},"vlan":{"burst":0,"subscribed":1,"using":1},"vlan_ecx":{"burst":0,"subscribed":0,"using":0},"zadara":{"burst":0,"subscribed":0,"using":0}}}`,
		`{"balance":{"balance":"155367.99237092264288897986","currency":"EUR","future":null},"usage":{"cpu":{"using":9007199254740993,"burst":0,"subscribed":10000},"future_resource":{"amount":9007199254740993.0000000001,"order":[3,2,1]}},"extension":null}`,
		`{}`, `{"balance":null}`, `{"usage":{}}`,
	}
	for _, body := range fixtures {
		calls := 0
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			calls++
			if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/currentusage/" {
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
		a, r, e := c.CurrentUsage.GetRaw(context.Background())
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
		(*a)["injected"] = json.RawMessage(`true`)
		if raw := (*a)["usage"]; len(raw) > 0 {
			raw[0] = 'X'
			if (*a)["usage"][0] != 'X' {
				t.Fatal("mutation failed")
			}
		}
		b, _, e := c.CurrentUsage.GetRaw(context.Background())
		if e != nil || calls != 2 || !reflect.DeepEqual(map[string]json.RawMessage(*b), expected) {
			t.Fatal("ownership", e)
		}
	}
}

func TestCurrentUsageErrors(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `1`, `"x"`, `{`, ``} {
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}))
		a, r, e := c.CurrentUsage.GetRaw(context.Background())
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
		a, r, e := c.CurrentUsage.GetRaw(context.Background())
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
	if _, _, e := c.CurrentUsage.GetRaw(ctx); !errors.Is(e, context.Canceled) {
		t.Fatal("cancel", e)
	}
}
