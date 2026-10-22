package cloudsigma_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type accountCreateCredentials struct{}

func (accountCreateCredentials) Retrieve() (sdk.Credentials, error) { return sdk.Credentials{}, nil }

type accountCreateWire struct {
	t                   *testing.T
	want                string
	body                string
	code, calls, closed int
	err                 error
}
type accountCreateBody struct {
	*strings.Reader
	w *accountCreateWire
}

func (b *accountCreateBody) Close() error { b.w.closed++; return nil }
func (w *accountCreateWire) RoundTrip(q *http.Request) (*http.Response, error) {
	w.calls++
	if q.Method != "POST" || q.URL.RequestURI() != "/custom/api/2.0/accounts/action/?do=create" {
		w.t.Fatalf("route %s %s", q.Method, q.URL)
	}
	if q.Header.Get("Authorization") != "" || q.Header.Get("Content-Type") != "application/json" || q.Header.Get("Accept") != "application/json" {
		w.t.Fatal("headers")
	}
	b, e := io.ReadAll(q.Body)
	if e != nil || strings.TrimSpace(string(b)) != w.want {
		w.t.Fatalf("body %s want %s", b, w.want)
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if w.err != nil {
		return nil, w.err
	}
	return &http.Response{StatusCode: w.code, Header: make(http.Header), Body: &accountCreateBody{strings.NewReader(w.body), w}, Request: q}, nil
}
func accountCreateClient(w *accountCreateWire) *sdk.Client {
	return sdk.NewClient(accountCreateCredentials{}, sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
}
func TestAccountCreatePresenceAndRaw(t *testing.T) {
	empty, promo := "", "synthetic-promo"
	for _, tc := range []struct {
		in   sdk.AccountCreateRequest
		want string
	}{
		{sdk.AccountCreateRequest{Email: "synthetic+42@example.invalid"}, `{"email":"synthetic+42@example.invalid"}`},
		{sdk.AccountCreateRequest{Email: "", Promo: &empty}, `{"email":"","promo":""}`},
		{sdk.AccountCreateRequest{Email: "not locally validated", Promo: &promo}, `{"email":"not locally validated","promo":"synthetic-promo"}`},
	} {
		body := `{"success":true,"uuid":"synthetic-account","unknown":{"n":9007199254740993,"decimal":1.2300,"nested":[null,false,{"k":"v"}]}}`
		w := &accountCreateWire{t: t, want: tc.want, body: body, code: 200}
		c := accountCreateClient(w)
		a, r, e := c.CreateAccountRaw(context.Background(), tc.in)
		var want map[string]json.RawMessage
		if err := json.Unmarshal([]byte(body), &want); err != nil {
			t.Fatal(err)
		}
		if e != nil || r == nil || r.StatusCode != 200 || a == nil || !reflect.DeepEqual(map[string]json.RawMessage(*a), want) || w.calls != 1 || w.closed != 1 {
			t.Fatalf("result %v %v %v", a, r, e)
		}
		(*a)["unknown"][0] = 'x'
		b, _, e := c.CreateAccountRaw(context.Background(), tc.in)
		if e != nil || !reflect.DeepEqual(map[string]json.RawMessage(*b), want) || w.calls != 2 || w.closed != 2 {
			t.Fatal("ownership", e)
		}
		if promo != "synthetic-promo" || empty != "" {
			t.Fatal("caller input changed")
		}
	}
}
func TestAccountCreateResponseErrors(t *testing.T) {
	for _, body := range []string{"", "null", "[]", "true", "42", `"text"`, `{"bad":`} {
		w := &accountCreateWire{t: t, want: `{"email":""}`, body: body, code: 200}
		a, r, e := accountCreateClient(w).CreateAccountRaw(context.Background(), sdk.AccountCreateRequest{})
		if e == nil || a != nil || r != nil || w.calls != 1 || w.closed != 1 {
			t.Fatalf("decode %q %v", body, e)
		}
	}
	for _, code := range []int{302, 400, 401, 403, 404, 409, 429, 500} {
		w := &accountCreateWire{t: t, want: `{"email":""}`, body: `[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`, code: code}
		a, r, e := accountCreateClient(w).CreateAccountRaw(context.Background(), sdk.AccountCreateRequest{})
		var api *sdk.ErrorResponse
		if a != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0].Message != "synthetic refusal" || w.calls != 1 || w.closed != 1 {
			t.Fatalf("HTTP %d %v", code, e)
		}
	}
}
func TestAccountCreateTransportAndSetup(t *testing.T) {
	sentinel := errors.New("synthetic transport")
	w := &accountCreateWire{t: t, want: `{"email":""}`, err: sentinel}
	a, r, e := accountCreateClient(w).CreateAccountRaw(context.Background(), sdk.AccountCreateRequest{})
	if a != nil || r != nil || !errors.Is(e, sentinel) || w.calls != 1 {
		t.Fatal("transport", e)
	}
	for _, endpoint := range []string{"http://example.invalid/", "https://example.invalid/api/?x=1"} {
		w = &accountCreateWire{t: t}
		c := sdk.NewClient(accountCreateCredentials{}, sdk.WithEndpoint(endpoint), sdk.WithHTTPClient(&http.Client{Transport: w}))
		a, r, e = c.CreateAccountRaw(context.Background(), sdk.AccountCreateRequest{})
		if a != nil || r != nil || e == nil || w.calls != 0 {
			t.Fatal("endpoint", e)
		}
	}
	for _, deadline := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		if deadline {
			cancel()
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		} else {
			cancel()
		}
		w = &accountCreateWire{t: t, want: `{"email":""}`}
		a, r, e = accountCreateClient(w).CreateAccountRaw(ctx, sdk.AccountCreateRequest{})
		cancel()
		want := context.Canceled
		if deadline {
			want = context.DeadlineExceeded
		}
		if a != nil || r != nil || !errors.Is(e, want) || w.calls > 1 {
			t.Fatal("context", e)
		}
	}
}
func TestAccountCreateDecoderAtomic(t *testing.T) {
	value := sdk.AccountCreateResult{"old": json.RawMessage(`true`)}
	for _, body := range []string{"null", "[]", `{"x":`} {
		if e := json.Unmarshal([]byte(body), &value); e == nil || string(value["old"]) != "true" {
			t.Fatal("atomic decoder", e)
		}
	}
	if e := json.Unmarshal([]byte(`{"new":false}`), &value); e != nil || value["old"] != nil || string(value["new"]) != "false" {
		t.Fatal("replacement", e)
	}
}
