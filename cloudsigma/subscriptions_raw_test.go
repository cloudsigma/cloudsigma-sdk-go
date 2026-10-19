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

func TestSubscriptionsRawListRaw(t *testing.T) {
	// The first fixture is the complete public populated listing. Synthetic
	// fixtures below test preservation only, not provider descendants/discount schema.
	public, err := os.ReadFile("testdata/subscriptions_public.json")
	if err != nil {
		t.Fatal(err)
	}
	fixtures := []string{string(public),
		`{"meta":{"limit":2,"offset":3,"total_count":9,"future":9007199254740993},"objects":[{"descendants":[{"opaque":[3,2,1]}],"discounts":{"synthetic":0.00000000000000000001},"discount_amount":null,"was_billed":false},{"future":9007199254740993.0000000001}],"price":"0E-20","extension":null}`,
		`{}`, `{"objects":null}`, `{"objects":[]}`, `{"meta":null}`,
	}
	for _, tc := range []struct {
		opts *SubscriptionListOptions
		uri  string
	}{
		{nil, "/api/2.0/subscriptions/"},
		{&SubscriptionListOptions{}, "/api/2.0/subscriptions/"},
		{&SubscriptionListOptions{Status: "active", Resources: []string{"cpu", "mem"}}, "/api/2.0/subscriptions/?resource=cpu%2Cmem&status=active"},
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
			a, r, e := c.Subscriptions.ListRaw(context.Background(), tc.opts)
			if e != nil || a == nil || r == nil || r.StatusCode != 200 || r.Meta != nil || calls != 1 {
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
			b, _, e := c.Subscriptions.ListRaw(context.Background(), tc.opts)
			if e != nil || calls != 2 || !reflect.DeepEqual(map[string]json.RawMessage(*b), expected) {
				t.Fatal("ownership", e)
			}
		}
	}
}

func TestSubscriptionsRawErrors(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `1`, `"x"`, `{`, ``} {
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}))
		a, r, e := c.Subscriptions.ListRaw(context.Background(), nil)
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
		a, r, e := c.Subscriptions.ListRaw(context.Background(), nil)
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
	if _, _, e := c.Subscriptions.ListRaw(ctx, nil); !errors.Is(e, context.Canceled) {
		t.Fatal("cancel", e)
	}
}

func TestSubscriptionsRawFilters(t *testing.T) {
	for _, status := range []string{"", "active", "inactive", "expired", "all", "notexpired"} {
		calls := 0
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			calls++
			want := "/api/2.0/subscriptions/?resource=dssd%2Ccpu%2Cmem%2Ctx%2Cip%2Cvlan"
			if status != "" {
				want += "&status=" + status
			}
			if q.Method != "GET" || q.URL.RequestURI() != want {
				t.Fatal("wire", q.URL)
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[]}`)), Request: q}, nil
		})}))
		if _, _, err := c.Subscriptions.ListRaw(context.Background(), &SubscriptionListOptions{Status: status, Resources: []string{"dssd", "cpu", "mem", "tx", "ip", "vlan"}}); err != nil || calls != 1 {
			t.Fatal(err, calls)
		}
	}
	calls := 0
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected dispatch") })}))
	for _, opts := range []*SubscriptionListOptions{{Status: "ACTIVE"}, {Status: "active&x=1"}, {Resources: []string{""}}, {Resources: []string{"cpu,mem"}}, {Resources: []string{"cpu", "unknown"}}} {
		a, r, e := c.Subscriptions.ListRaw(context.Background(), opts)
		if a != nil || r != nil || e == nil || calls != 0 {
			t.Fatal("invalid dispatched", e, calls)
		}
	}
}

func TestSubscriptionsRawDecoderOwnership(t *testing.T) {
	input := []byte(`{"objects":[{"descendants":[],"discounts":null}],"price":"0E-20"}`)
	var value SubscriptionsRaw
	if err := json.Unmarshal(input, &value); err != nil {
		t.Fatal(err)
	}
	before := string(value["objects"])
	for i := range input {
		input[i] = 'X'
	}
	if input[0] != 'X' || string(value["objects"]) != before {
		t.Fatal("input ownership")
	}
	snapshot, _ := json.Marshal(value)
	for _, invalid := range []string{`null`, `[]`, `{`} {
		if err := value.UnmarshalJSON([]byte(invalid)); err == nil {
			t.Fatal("accepted invalid")
		}
		after, _ := json.Marshal(value)
		if string(after) != string(snapshot) {
			t.Fatal("failure mutated receiver")
		}
	}
}

func TestSubscriptionsRawOracleMutations(t *testing.T) {
	// Negative controls prove the raw comparison detects missing/null changes,
	// precision loss, envelope/type changes and array reorder without float64.
	original := `{"meta":null,"objects":[1,2],"n":9007199254740993.0000000001}`
	var expected map[string]json.RawMessage
	if err := json.Unmarshal([]byte(original), &expected); err != nil {
		t.Fatal(err)
	}
	for _, changed := range []string{
		`{"objects":[1,2],"n":9007199254740993.0000000001}`,
		`{"meta":{},"objects":[1,2],"n":9007199254740993.0000000001}`,
		`{"meta":null,"objects":[1,2],"n":9007199254740992}`,
		`{"meta":null,"objects":{},"n":9007199254740993.0000000001}`,
		`{"meta":null,"objects":[2,1],"n":9007199254740993.0000000001}`,
	} {
		var actual map[string]json.RawMessage
		if err := json.Unmarshal([]byte(changed), &actual); err != nil {
			t.Fatal(err)
		}
		if reflect.DeepEqual(actual, expected) {
			t.Fatal("oracle missed mutation")
		}
	}
}
