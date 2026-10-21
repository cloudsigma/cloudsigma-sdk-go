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

type ledgerPageWire struct {
	t                   *testing.T
	query, body         string
	code, calls, closed int
	err                 error
}
type ledgerPageBody struct {
	*strings.Reader
	wire *ledgerPageWire
}

func (b *ledgerPageBody) Close() error { b.wire.closed++; return nil }
func (w *ledgerPageWire) RoundTrip(q *http.Request) (*http.Response, error) {
	w.calls++
	if q.Body != nil {
		b, err := io.ReadAll(q.Body)
		if err != nil || len(b) != 0 {
			w.t.Fatal("unexpected request payload", string(b), err)
		}
	}
	if q.Method != "GET" || q.URL.Path != "/custom/api/2.0/ledger/" || q.URL.RawQuery != w.query {
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
	return &http.Response{StatusCode: w.code, Header: make(http.Header), Body: &ledgerPageBody{strings.NewReader(w.body), w}, Request: q}, nil
}
func ledgerPageClient(w *ledgerPageWire) *sdk.Client {
	return sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
}
func TestLedgerPageContract(t *testing.T) {
	for _, tc := range []struct {
		name  string
		opts  *sdk.ListOptions
		query string
	}{
		{"nil", nil, ""}, {"all", &sdk.ListOptions{}, "limit=0"}, {"page", &sdk.ListOptions{Limit: 2, Offset: 3}, "limit=2&offset=3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"meta":{"limit":2,"offset":3,"total_count":6,"next":"https://untrusted.invalid/"},"objects":[{"id":"39509304","amount":"0.00014583333333333333","initial":"469291.07502821435786823786","end":"469291.07488238102453490453","sequence":9007199254740993,"nullable":null,"unknown":{"number":1.2300}}],"future":null}`
			w := &ledgerPageWire{t: t, query: tc.query, body: body, code: 200}
			var before sdk.ListOptions
			if tc.opts != nil {
				before = *tc.opts
			}
			got, resp, err := ledgerPageClient(w).Ledger.GetPage(context.Background(), tc.opts)
			if err != nil || resp == nil || resp.StatusCode != 200 || got == nil || w.calls != 1 || w.closed != 1 {
				t.Fatalf("result %v %v %v %+v", got, resp, err, w)
			}
			if string((*got)["objects"]) != `[{"id":"39509304","amount":"0.00014583333333333333","initial":"469291.07502821435786823786","end":"469291.07488238102453490453","sequence":9007199254740993,"nullable":null,"unknown":{"number":1.2300}}]` || string((*got)["future"]) != "null" || string((*got)["meta"]) != `{"limit":2,"offset":3,"total_count":6,"next":"https://untrusted.invalid/"}` {
				t.Fatal("raw envelope changed", got)
			}
			if tc.opts != nil && !reflect.DeepEqual(*tc.opts, before) {
				t.Fatal("mutated options")
			}
			// Each request owns its response bytes, while ordinary map copies may alias.
			(*got)["objects"][0] = 'x'
			again, _, err := ledgerPageClient(w).Ledger.GetPage(context.Background(), tc.opts)
			if err != nil || (*again)["objects"][0] != '[' || w.calls != 2 {
				t.Fatal("response ownership", err)
			}
		})
	}
}
func TestLedgerPageInvalid(t *testing.T) {
	for _, o := range []sdk.ListOptions{{Limit: -1}, {Offset: -1}, {Limit: -1, Offset: -1}} {
		w := &ledgerPageWire{t: t}
		v, r, e := ledgerPageClient(w).Ledger.GetPage(context.Background(), &o)
		if e == nil || v != nil || r != nil || w.calls != 0 {
			t.Fatal("invalid pagination dispatched")
		}
	}
}
func TestLedgerPageErrors(t *testing.T) {
	for _, code := range []int{401, 403, 404, 409, 429, 500} {
		w := &ledgerPageWire{t: t, code: code, body: `[{"error_message":"denied","error_type":"synthetic","error_point":"ledger"}]`}
		v, r, e := ledgerPageClient(w).Ledger.GetPage(context.Background(), nil)
		var api *sdk.ErrorResponse
		if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || w.closed != 1 {
			t.Fatalf("error %d: %v %v %v", code, v, r, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	w := &ledgerPageWire{t: t, err: sentinel}
	v, r, e := ledgerPageClient(w).Ledger.GetPage(context.Background(), nil)
	if v != nil || r != nil || !errors.Is(e, sentinel) {
		t.Fatal("transport identity", e)
	}
	for _, body := range []string{`[]`, `null`, `{`} {
		w = &ledgerPageWire{t: t, code: 200, body: body}
		v, r, e = ledgerPageClient(w).Ledger.GetPage(context.Background(), nil)
		if v != nil || r != nil || e == nil || w.closed != 1 {
			t.Fatal("shape accepted", body, e)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w = &ledgerPageWire{t: t}
	v, r, e = ledgerPageClient(w).Ledger.GetPage(ctx, nil)
	if v != nil || r != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("cancellation", e)
	}
}
func TestLedgerPageLegacy(t *testing.T) {
	w := &ledgerPageWire{t: t, code: 200, body: `{"objects":[],"meta":{"limit":20}}`}
	c := ledgerPageClient(w)
	a, _, e := c.Ledger.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	b, _, e := c.Ledger.GetPage(context.Background(), nil)
	if e != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("nil compatibility", e)
	}
}

var _ io.ReadCloser = (*ledgerPageBody)(nil)
