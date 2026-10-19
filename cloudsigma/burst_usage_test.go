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

func TestBurstUsageListRaw(t *testing.T) {
	fixtures := []string{
		// Public burst example with only its invalid final trailing comma removed.
		`{"objects":[{"uuid":"75fbf5d6-4ef6-11e4-9ce3-8bd2cacc5639","amount":0.0100,"resource_type":"ip","usage":1},{"uuid":"f2669d85-2dbd-4f0d-8a1f-40a06deb3233","amount":0.0301,"resource_type":"cpu","usage":12000},{"uuid":"1e170e6e-5339-477d-a29d-c3e6533a1825","amount":0.0200,"resource_type":"mem","usage":9663676416},{"uuid":"42b2af55-ec3c-4066-a152-66d201c59576","amount":0.0100,"resource_type":"dssd","usage":1073741824000}]}`,
		`{"meta":{"future":9007199254740993},"objects":[{"amount":"0.0000000000000000001","usage":9007199254740993.0000000001,"optional":null,"future":[3,2,1]}],"extension":null}`,
		`{}`, `{"objects":null}`, `{"objects":[]}`, `{"meta":null}`,
	}
	for _, tc := range []struct {
		opts *BurstUsageListOptions
		uri  string
	}{
		{nil, "/api/2.0/burstusage/"},
		{&BurstUsageListOptions{Date: "2014-10-21"}, "/api/2.0/burstusage/?date=2014-10-21"},
		{&BurstUsageListOptions{DateLTE: "2014-10-21"}, "/api/2.0/burstusage/?date__lte=2014-10-21"},
		{&BurstUsageListOptions{DateGTE: "2014-07-01", DateLT: "2014-08-01"}, "/api/2.0/burstusage/?date__gte=2014-07-01&date__lt=2014-08-01"},
		{&BurstUsageListOptions{Date: "a", DateGT: "b", DateGTE: "c", DateLT: "d", DateLTE: "e"}, "/api/2.0/burstusage/?date=a&date__gt=b&date__gte=c&date__lt=d&date__lte=e"},
		{&BurstUsageListOptions{}, "/api/2.0/burstusage/"},
		{&BurstUsageListOptions{DateGT: "2014-11-15", DateLT: "2014-11-19"}, "/api/2.0/burstusage/?date__gt=2014-11-15&date__lt=2014-11-19"},
		{&BurstUsageListOptions{ListOptions: &ListOptions{Limit: 0}}, "/api/2.0/burstusage/?limit=0"},
		{&BurstUsageListOptions{DateGT: "2014-07-01", DateLT: "2014-08-01", ListOptions: &ListOptions{Limit: 2, Offset: 3}}, "/api/2.0/burstusage/?date__gt=2014-07-01&date__lt=2014-08-01&limit=2&offset=3"},
		{&BurstUsageListOptions{DateGT: "synthetic +&value"}, "/api/2.0/burstusage/?date__gt=synthetic+%2B%26value"},
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
			a, r, e := c.BurstUsage.ListRaw(context.Background(), tc.opts)
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
			b, _, e := c.BurstUsage.ListRaw(context.Background(), tc.opts)
			if e != nil || calls != 2 || !reflect.DeepEqual(map[string]json.RawMessage(*b), expected) {
				t.Fatal("ownership", e)
			}
		}
	}
}

func TestBurstUsageErrors(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `1`, `"x"`, `{`, ``} {
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}))
		a, r, e := c.BurstUsage.ListRaw(context.Background(), nil)
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
		a, r, e := c.BurstUsage.ListRaw(context.Background(), nil)
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
	if _, _, e := c.BurstUsage.ListRaw(ctx, nil); !errors.Is(e, context.Canceled) {
		t.Fatal("cancel", e)
	}
}
