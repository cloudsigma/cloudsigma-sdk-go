package cloudsigma_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type bserverGetWire struct {
	t                   *testing.T
	body                string
	code, calls, closed int
	err                 error
}
type bserverGetBody struct {
	*strings.Reader
	w *bserverGetWire
}

func (b *bserverGetBody) Close() error { b.w.closed++; return nil }
func (w *bserverGetWire) RoundTrip(q *http.Request) (*http.Response, error) {
	w.calls++
	if q.Method != "GET" || q.URL.RequestURI() != "/custom/api/2.0/bservers/synthetic-server/" {
		w.t.Fatalf("route %s %s", q.Method, q.URL)
	}
	if q.Header.Get("Authorization") != "Bearer synthetic" {
		w.t.Fatal("authentication")
	}
	if q.Body != nil {
		b, e := io.ReadAll(q.Body)
		if e != nil || len(b) != 0 {
			w.t.Fatal("request body")
		}
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if w.err != nil {
		return nil, w.err
	}
	return &http.Response{StatusCode: w.code, Header: make(http.Header), Body: &bserverGetBody{strings.NewReader(w.body), w}, Request: q}, nil
}
func bserverGetClient(w *bserverGetWire) *sdk.Client {
	return sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
}
func TestBServersGetRawContract(t *testing.T) {
	w := &bserverGetWire{t: t, code: 200, body: `{"uuid":"synthetic-server","runtime":{"perf_factor":1.2300,"nics":[{"traffic":9007199254740993,"dynamic_ip":null}]},"unknown":{"nested":[null,9007199254740993]},"nullable":null}`}
	c := bserverGetClient(w)
	got, r, e := c.BServers.GetRaw(context.Background(), "synthetic-server")
	if e != nil || r == nil || r.StatusCode != 200 || r.Meta != nil || w.calls != 1 || w.closed != 1 {
		t.Fatalf("result %v %v %+v", r, e, w)
	}
	if string(got["runtime"]) != `{"perf_factor":1.2300,"nics":[{"traffic":9007199254740993,"dynamic_ip":null}]}` || string(got["unknown"]) != `{"nested":[null,9007199254740993]}` || string(got["nullable"]) != "null" {
		t.Fatal("raw preservation", got)
	}
	got["runtime"][0] = '!'
	again, _, e := c.BServers.GetRaw(context.Background(), "synthetic-server")
	if e != nil || again["runtime"][0] != '{' || w.calls != 2 || w.closed != 2 {
		t.Fatal("ownership", e)
	}
	w.body = `{}`
	got, _, e = c.BServers.GetRaw(context.Background(), "synthetic-server")
	if e != nil || got == nil || len(got) != 0 {
		t.Fatal("empty object", e)
	}
}
func TestBServersGetRawRefusals(t *testing.T) {
	for _, id := range []string{"", ".", "..", "a/b", "a?x", "a#x", "%2f", "a\n"} {
		w := &bserverGetWire{t: t}
		v, r, e := bserverGetClient(w).BServers.GetRaw(context.Background(), id)
		if e == nil || v != nil || r != nil || w.calls != 0 {
			t.Fatalf("invalid identifier %q", id)
		}
	}
	for _, body := range []string{`null`, `[]`, `true`, `1`, `"text"`, `{`} {
		w := &bserverGetWire{t: t, code: 200, body: body}
		v, r, e := bserverGetClient(w).BServers.GetRaw(context.Background(), "synthetic-server")
		if e == nil || v != nil || (body == `null` && r == nil) || (body != `null` && r != nil) || w.calls != 1 || w.closed != 1 {
			t.Fatalf("invalid response %s: %v", body, e)
		}
	}
}
func TestBServersGetRawErrors(t *testing.T) {
	for _, code := range []int{401, 403, 404, 409, 429, 500} {
		w := &bserverGetWire{t: t, code: code, body: `[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`}
		v, r, e := bserverGetClient(w).BServers.GetRaw(context.Background(), "synthetic-server")
		var api *sdk.ErrorResponse
		if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0].Message != "synthetic refusal" || w.closed != 1 {
			t.Fatalf("HTTP%d: %v", code, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	w := &bserverGetWire{t: t, err: sentinel}
	v, r, e := bserverGetClient(w).BServers.GetRaw(context.Background(), "synthetic-server")
	if v != nil || r != nil || !errors.Is(e, sentinel) {
		t.Fatal("transport", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w = &bserverGetWire{t: t}
	v, _, e = bserverGetClient(w).BServers.GetRaw(ctx, "synthetic-server")
	if v != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("context", e)
	}
}
