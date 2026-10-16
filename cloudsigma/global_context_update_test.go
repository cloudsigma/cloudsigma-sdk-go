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

func TestGlobalContextUpdate(t *testing.T) {
	for _, input := range []map[string]string{{"new_global_key": "new_global_val"}, {}, {"keep": "existing", "unicode": "☃", "empty": ""}} {
		calls := 0
		// First fixture matches the public POST example; extra response members are
		// synthetic retention probes, not authorization to submit non-string values.
		body := `{"new_global_key":"new_global_val","future":{"n":9007199254740993},"nullable":null}`
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			calls++
			if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/global_context/" {
				t.Fatal("wire", q.Method, q.URL.RequestURI())
			}
			b, e := io.ReadAll(q.Body)
			if e != nil {
				t.Fatal(e)
			}
			var got map[string]string
			if e = json.Unmarshal(b, &got); e != nil || !reflect.DeepEqual(got, input) {
				t.Fatalf("direct request: %s %v", b, e)
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}))
		value, r, e := c.GlobalContext.Update(context.Background(), input)
		if e != nil || r == nil || r.StatusCode != 200 || value == nil || calls != 1 {
			t.Fatal("update", e, calls)
		}
		var want map[string]json.RawMessage
		if e = json.Unmarshal([]byte(body), &want); e != nil {
			t.Fatal(e)
		}
		if !reflect.DeepEqual(map[string]json.RawMessage(*value), want) {
			t.Fatal("raw response")
		}
		(*value)["future"][0] = 'X'
		if (*value)["future"][0] != 'X' {
			t.Fatal("mutation witness")
		}
		second, _, e := c.GlobalContext.Update(context.Background(), input)
		if e != nil || calls != 2 || !reflect.DeepEqual(map[string]json.RawMessage(*second), want) {
			t.Fatal("independent result", e)
		}
	}
}

func TestGlobalContextUpdateRefusals(t *testing.T) {
	calls := 0
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected dispatch") })}))
	for _, input := range []map[string]string{nil, {"\xff": "value"}, {"key": "\xff"}} {
		v, r, e := c.GlobalContext.Update(context.Background(), input)
		if e == nil || v != nil || r != nil || calls != 0 {
			t.Fatal("local refusal", e, calls)
		}
	}
}

func TestGlobalContextUpdateErrors(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `1`, `"x"`, `{`, ``} {
		calls := 0
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}))
		v, r, e := c.GlobalContext.Update(context.Background(), map[string]string{"key": "value"})
		if e == nil || v != nil || r != nil || calls != 1 {
			t.Fatal("malformed", body, e, calls)
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
		v, r, e := c.GlobalContext.Update(context.Background(), map[string]string{"key": "value"})
		if v != nil || e == nil || calls != 1 {
			t.Fatal("error", e, calls)
		}
		if code == 0 {
			if !errors.Is(e, sentinel) || r != nil {
				t.Fatal("transport", e)
			}
		} else {
			var api *ErrorResponse
			if !errors.As(e, &api) || r == nil || r.StatusCode != code || len(api.Errors) != 1 || api.Errors[0].Message != "synthetic denied" || api.Errors[0].Type != "permission" || api.Response != r {
				t.Fatal("typed", e)
			}
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) { calls++; return nil, q.Context().Err() })}))
	if v, r, e := c.GlobalContext.Update(ctx, map[string]string{"key": "value"}); !errors.Is(e, context.Canceled) || v != nil || r != nil || calls != 1 {
		t.Fatal("cancel", e, calls)
	}
}
