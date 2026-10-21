package cloudsigma_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type subscriptionPageWire struct {
	t                   *testing.T
	query, body         string
	code, calls, closed int
	err                 error
}
type subscriptionPageBody struct {
	*strings.Reader
	wire *subscriptionPageWire
}

func (b *subscriptionPageBody) Close() error { b.wire.closed++; return nil }
func (w *subscriptionPageWire) RoundTrip(q *http.Request) (*http.Response, error) {
	w.calls++
	if q.Body != nil {
		b, err := io.ReadAll(q.Body)
		if err != nil || len(b) != 0 {
			w.t.Fatal("unexpected request payload", string(b), err)
		}
	}
	if q.Method != "GET" || q.URL.Path != "/custom/api/2.0/subscriptions/" || q.URL.RawQuery != w.query {
		w.t.Fatalf("request: %s %s body=%v", q.Method, q.URL, q.Body)
	}
	if q.Header.Get("Authorization") != "Bearer synthetic" {
		w.t.Fatal("missing authentication")
	}
	if err := q.Context().Err(); err != nil {
		return nil, err
	}
	if w.err != nil {
		return nil, w.err
	}
	return &http.Response{StatusCode: w.code, Header: make(http.Header), Body: &subscriptionPageBody{strings.NewReader(w.body), w}, Request: q}, nil
}
func subscriptionPageClient(w *subscriptionPageWire) *sdk.Client {
	return sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
}
func TestSubscriptionPageContract(t *testing.T) {
	for _, tc := range []struct {
		name  string
		opts  *sdk.ListOptions
		query string
	}{
		{"nil", nil, ""}, {"all", &sdk.ListOptions{}, "limit=0"}, {"page", &sdk.ListOptions{Limit: 2, Offset: 3}, "limit=2&offset=3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"meta":{"limit":2,"offset":3,"total_count":6,"next":"https://untrusted.invalid/"},"objects":[{"id":"14","amount":"4096","price":"0.00014583333333333333","resource":"mem","descendants":[{"discount":"0.123456789012345678"}],"sequence":9007199254740993,"nullable":null,"unknown":{"number":1.2300}}],"future":null}`
			w := &subscriptionPageWire{t: t, query: tc.query, body: body, code: 200}
			var before sdk.ListOptions
			if tc.opts != nil {
				before = *tc.opts
			}
			got, resp, err := subscriptionPageClient(w).Subscriptions.ListRawPage(context.Background(), nil, tc.opts)
			if err != nil || resp == nil || resp.StatusCode != 200 || got == nil || w.calls != 1 || w.closed != 1 {
				t.Fatalf("result %v %v %v %+v", got, resp, err, w)
			}
			if string((*got)["objects"]) != `[{"id":"14","amount":"4096","price":"0.00014583333333333333","resource":"mem","descendants":[{"discount":"0.123456789012345678"}],"sequence":9007199254740993,"nullable":null,"unknown":{"number":1.2300}}]` || string((*got)["future"]) != "null" || string((*got)["meta"]) != `{"limit":2,"offset":3,"total_count":6,"next":"https://untrusted.invalid/"}` {
				t.Fatal("raw envelope changed", got)
			}
			if tc.opts != nil && !reflect.DeepEqual(*tc.opts, before) {
				t.Fatal("mutated options")
			}
			// Each request owns its response bytes, while ordinary map copies may alias.
			(*got)["objects"][0] = 'x'
			again, _, err := subscriptionPageClient(w).Subscriptions.ListRawPage(context.Background(), nil, tc.opts)
			if err != nil || (*again)["objects"][0] != '[' || w.calls != 2 {
				t.Fatal("response ownership", err)
			}
		})
	}
}
func TestSubscriptionPageInvalid(t *testing.T) {
	for _, o := range []sdk.ListOptions{{Limit: -1}, {Offset: -1}, {Limit: -1, Offset: -1}} {
		w := &subscriptionPageWire{t: t}
		v, r, e := subscriptionPageClient(w).Subscriptions.ListRawPage(context.Background(), nil, &o)
		if e == nil || v != nil || r != nil || w.calls != 0 {
			t.Fatal("invalid pagination dispatched")
		}
	}
}
func TestSubscriptionPageErrors(t *testing.T) {
	for _, code := range []int{401, 403, 404, 409, 429, 500} {
		w := &subscriptionPageWire{t: t, code: code, body: `[{"error_message":"denied","error_type":"synthetic","error_point":"subscriptions"}]`}
		v, r, e := subscriptionPageClient(w).Subscriptions.ListRawPage(context.Background(), nil, nil)
		var api *sdk.ErrorResponse
		if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || w.closed != 1 {
			t.Fatalf("error %d: %v %v %v", code, v, r, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	w := &subscriptionPageWire{t: t, err: sentinel}
	v, r, e := subscriptionPageClient(w).Subscriptions.ListRawPage(context.Background(), nil, nil)
	if v != nil || r != nil || !errors.Is(e, sentinel) {
		t.Fatal("transport identity", e)
	}
	for _, body := range []string{`[]`, `null`, `{`} {
		w = &subscriptionPageWire{t: t, code: 200, body: body}
		v, r, e = subscriptionPageClient(w).Subscriptions.ListRawPage(context.Background(), nil, nil)
		if v != nil || r != nil || e == nil || w.closed != 1 {
			t.Fatal("shape accepted", body, e)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w = &subscriptionPageWire{t: t}
	v, r, e = subscriptionPageClient(w).Subscriptions.ListRawPage(ctx, nil, nil)
	if v != nil || r != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("cancellation", e)
	}
}
func TestSubscriptionPageLegacy(t *testing.T) {
	w := &subscriptionPageWire{t: t, code: 200, body: `{"objects":[],"meta":{"limit":20}}`}
	c := subscriptionPageClient(w)
	a, _, e := c.Subscriptions.ListRaw(context.Background(), nil)
	if e != nil {
		t.Fatal(e)
	}
	b, _, e := c.Subscriptions.ListRawPage(context.Background(), nil, nil)
	if e != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("nil compatibility", e)
	}
}

var _ io.ReadCloser = (*subscriptionPageBody)(nil)

func TestSubscriptionPageFilters(t *testing.T) {
	filters := &sdk.SubscriptionListOptions{Status: "active", Resources: []string{"cpu", "mem"}}
	before := sdk.SubscriptionListOptions{Status: "active", Resources: []string{"cpu", "mem"}}
	for _, tc := range []struct {
		page  *sdk.ListOptions
		query string
	}{
		{nil, "resource=cpu%2Cmem&status=active"},
		{&sdk.ListOptions{}, "limit=0&resource=cpu%2Cmem&status=active"},
		{&sdk.ListOptions{Limit: 2, Offset: 3}, "limit=2&offset=3&resource=cpu%2Cmem&status=active"},
	} {
		w := &subscriptionPageWire{t: t, code: 200, body: `{"objects":[],"meta":{}}`, query: tc.query}
		got, r, e := subscriptionPageClient(w).Subscriptions.ListRawPage(context.Background(), filters, tc.page)
		if e != nil || got == nil || r == nil || r.Meta != nil || !reflect.DeepEqual(*filters, before) {
			t.Fatal("combined filters", got, r, e, filters)
		}
	}
	for _, f := range []*sdk.SubscriptionListOptions{{Status: "invalid"}, {Resources: []string{"cpu", "invalid"}}} {
		w := &subscriptionPageWire{t: t}
		got, r, e := subscriptionPageClient(w).Subscriptions.ListRawPage(context.Background(), f, &sdk.ListOptions{Limit: 2})
		if e == nil || got != nil || r != nil || w.calls != 0 {
			t.Fatal("invalid filter dispatched")
		}
	}
}
