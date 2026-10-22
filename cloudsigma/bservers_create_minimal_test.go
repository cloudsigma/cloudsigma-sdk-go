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

type bserverMinimalCreateWire struct {
	t                   *testing.T
	body                string
	code, calls, closed int
	want                string
	err                 error
}
type bserverMinimalCreateBody struct {
	*strings.Reader
	w *bserverMinimalCreateWire
}

func (b *bserverMinimalCreateBody) Close() error { b.w.closed++; return nil }
func (w *bserverMinimalCreateWire) RoundTrip(q *http.Request) (*http.Response, error) {
	w.calls++
	if q.Method != "POST" || q.URL.RequestURI() != "/custom/api/2.0/bservers/" {
		w.t.Fatalf("route %s %s", q.Method, q.URL)
	}
	if q.Header.Get("Authorization") != "Bearer synthetic" {
		w.t.Fatal("authentication")
	}
	b, e := io.ReadAll(q.Body)
	if e != nil {
		w.t.Fatal(e)
	}
	var got, want interface{}
	if json.Unmarshal(b, &got) != nil || json.Unmarshal([]byte(w.want), &want) != nil || !reflect.DeepEqual(got, want) {
		w.t.Fatalf("body %s wanted %s", b, w.want)
	}
	if q.Header.Get("Content-Type") != "application/json" {
		w.t.Fatal("content type")
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if w.err != nil {
		return nil, w.err
	}
	return &http.Response{StatusCode: w.code, Header: make(http.Header), Body: &bserverMinimalCreateBody{strings.NewReader(w.body), w}, Request: q}, nil
}
func bserverMinimalCreateClient(w *bserverMinimalCreateWire) *sdk.Client {
	return sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
}
func TestBServersCreateMinimalEnvelope(t *testing.T) {
	for _, input := range [][]sdk.BServerMinimalCreateRequest{nil, {}, {{CPU: 1000, Mem: 536870912, Name: "Name", VNCPassword: "synthetic"}, {}}} {
		want := `{"objects":[]}`
		if len(input) > 0 {
			want = `{"objects":[{"cpu":1000,"mem":536870912,"name":"Name","vnc_password":"synthetic"},{"cpu":0,"mem":0,"name":"","vnc_password":""}]}`
		}
		body := `{"objects":[{"email":"a@example.invalid","main":null,"name":"Name","phone":"+1123","resource_uri":"/api/2.0/bservers/id/","uuid":"id","future":123456789012345678901}],"future":null}`
		w := &bserverMinimalCreateWire{t: t, code: 201, body: body, want: want}
		v, r, e := bserverMinimalCreateClient(w).BServers.CreateMinimalRaw(context.Background(), input)
		if e != nil || v == nil || r == nil || r.StatusCode != 201 || r.Meta != nil || w.calls != 1 || w.closed != 1 {
			t.Fatalf("result %v %v %v", v, r, e)
		}
		if string((*v)["objects"]) != `[{"email":"a@example.invalid","main":null,"name":"Name","phone":"+1123","resource_uri":"/api/2.0/bservers/id/","uuid":"id","future":123456789012345678901}]` || string((*v)["future"]) != "null" {
			t.Fatal("raw preservation", v)
		}
	}
}
func TestBServersCreateMinimalErrors(t *testing.T) {
	for _, code := range []int{400, 401, 403, 404, 409, 429, 500} {
		w := &bserverMinimalCreateWire{t: t, code: code, want: `{"objects":[]}`, body: `[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`}
		v, r, e := bserverMinimalCreateClient(w).BServers.CreateMinimalRaw(context.Background(), nil)
		var api *sdk.ErrorResponse
		if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0].Message != "synthetic refusal" || w.calls != 1 || w.closed != 1 {
			t.Fatalf("HTTP%d: %v", code, e)
		}
	}
	for _, body := range []string{`[]`, `null`, `{"objects":`} {
		w := &bserverMinimalCreateWire{t: t, code: 201, want: `{"objects":[]}`, body: body}
		v, r, e := bserverMinimalCreateClient(w).BServers.CreateMinimalRaw(context.Background(), nil)
		if v != nil || r != nil || e == nil || w.closed != 1 {
			t.Fatalf("invalid response %s %v", body, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	w := &bserverMinimalCreateWire{t: t, err: sentinel, want: `{"objects":[]}`}
	v, r, e := bserverMinimalCreateClient(w).BServers.CreateMinimalRaw(context.Background(), nil)
	if v != nil || r != nil || !errors.Is(e, sentinel) || w.calls != 1 {
		t.Fatal("transport", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w = &bserverMinimalCreateWire{t: t, want: `{"objects":[]}`}
	v, r, e = bserverMinimalCreateClient(w).BServers.CreateMinimalRaw(ctx, nil)
	if v != nil || r != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("context", e)
	}
}

func TestBServersCreateMinimalBoundaries(t *testing.T) {
	input := []sdk.BServerMinimalCreateRequest{{CPU: 1000, Mem: 536870912, Name: "雪\"\n", VNCPassword: "synthetic\\value"}}
	before := input[0]
	w := &bserverMinimalCreateWire{t: t, code: 201, want: `{"objects":[{"cpu":1000,"mem":536870912,"name":"雪\"\n","vnc_password":"synthetic\\value"}]}`, body: `{"objects":[{"runtime":{"perf_factor":1.2300,"nics":[{"traffic":9007199254740993,"ip":null}]}}],"unknown":{"nested":[null,9007199254740993]}}`}
	c := bserverMinimalCreateClient(w)
	v, _, e := c.BServers.CreateMinimalRaw(context.Background(), input)
	if e != nil || input[0] != before || string((*v)["unknown"]) != `{"nested":[null,9007199254740993]}` || string((*v)["objects"]) != `[{"runtime":{"perf_factor":1.2300,"nics":[{"traffic":9007199254740993,"ip":null}]}}]` {
		t.Fatal("preservation", e)
	}
	(*v)["objects"][0] = '!'
	again, _, e := c.BServers.CreateMinimalRaw(context.Background(), input)
	if e != nil || (*again)["objects"][0] != '[' || w.calls != 2 || w.closed != 2 {
		t.Fatal("independent results", e)
	}
	w = &bserverMinimalCreateWire{t: t}
	c = sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint(":bad"), sdk.WithHTTPClient(&http.Client{Transport: w}))
	v, r, e := c.BServers.CreateMinimalRaw(context.Background(), input)
	if e == nil || v != nil || r != nil || w.calls != 0 {
		t.Fatal("setup refusal")
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(1, 0))
	defer cancel()
	w = &bserverMinimalCreateWire{t: t, want: `{"objects":[]}`}
	v, _, e = bserverMinimalCreateClient(w).BServers.CreateMinimalRaw(ctx, nil)
	if v != nil || !errors.Is(e, context.DeadlineExceeded) || w.calls > 1 {
		t.Fatal("deadline", e)
	}
	w = &bserverMinimalCreateWire{t: t, code: 201, want: `{"objects":[]}`, body: ""}
	v, r, e = bserverMinimalCreateClient(w).BServers.CreateMinimalRaw(context.Background(), nil)
	if !errors.Is(e, io.EOF) || v != nil || r != nil || w.closed != 1 || w.calls != 1 {
		t.Fatal("shared empty-body convention", e)
	}
}
