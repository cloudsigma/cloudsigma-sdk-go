package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"
)

type calculatorBody struct {
	io.Reader
	closed int
}

func (b *calculatorBody) Close() error { b.closed++; return nil }

type calculatorWire struct {
	t           *testing.T
	want        string
	code, calls int
	body        *calculatorBody
	err         error
}

func (f *calculatorWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/subscriptioncalculator/" {
		f.t.Fatal("route/method")
	}
	b, e := io.ReadAll(q.Body)
	if e != nil {
		f.t.Fatal(e)
	}
	// UseNumber preserves numeric tokens; maps ignore irrelevant object-key order.
	decode := func(b []byte) interface{} {
		d := json.NewDecoder(bytes.NewReader(b))
		d.UseNumber()
		var v interface{}
		if e := d.Decode(&v); e != nil {
			f.t.Fatal(e)
		}
		return v
	}
	if !reflect.DeepEqual(decode(b), decode([]byte(f.want))) {
		f.t.Fatalf("request %s", b)
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: f.body, Request: q}, nil
}
func calculatorClient(f *calculatorWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}

const calculatorDefinition = `{"amount":10000000000,"period":"1 month","resource":"dssd"}`
const calculatorRequest = `{"objects":[` + calculatorDefinition + `]}`

func TestSubscriptionCalculatePublic(t *testing.T) {
	b := &calculatorBody{Reader: bytes.NewBufferString(`{"objects":[{"amount":"10000000000","price":"1.010484993457794189453125","discounts":{},"free_tier":false}],"price":"1.010484993457794189453125"}`)}
	f := &calculatorWire{t: t, want: calculatorRequest, code: 200, body: b}
	out, r, e := calculatorClient(f).Subscriptions.Calculate(context.Background(), []json.RawMessage{json.RawMessage(calculatorDefinition)})
	if e != nil || r == nil || r.StatusCode != 200 || f.calls != 1 || b.closed != 1 || string(out["price"]) != `"1.010484993457794189453125"` || string(out["objects"]) != `[{"amount":"10000000000","price":"1.010484993457794189453125","discounts":{},"free_tier":false}]` {
		t.Fatalf("out=%s r=%v e=%v", out, r, e)
	}
}
func TestSubscriptionCalculatePresence(t *testing.T) {
	// Synthetic fields prove forwarding, not server acceptance or valid prices.
	defs := []json.RawMessage{json.RawMessage(`{"amount":9007199254740993,"flag":false,"empty":"","nil":null,"items":[2,1]}`), json.RawMessage(`{"amount":0}`)}
	before := append([]byte(nil), defs[0]...)
	b := &calculatorBody{Reader: bytes.NewBufferString(`{"objects":[],"price":"0","extension":9007199254740993}`)}
	f := &calculatorWire{t: t, want: `{"objects":[` + string(defs[0]) + `,` + string(defs[1]) + `]}`, code: 200, body: b}
	out, _, e := calculatorClient(f).Subscriptions.Calculate(context.Background(), defs)
	if e != nil || f.calls != 1 || b.closed != 1 || !bytes.Equal(before, defs[0]) || string(out["extension"]) != "9007199254740993" {
		t.Fatal(out, e)
	}
}
func TestSubscriptionCalculateRefusals(t *testing.T) {
	for _, defs := range [][]json.RawMessage{nil, {}, {nil}, {json.RawMessage(`null`)}, {json.RawMessage(`[]`)}, {json.RawMessage(`1`)}, {json.RawMessage(`{`)}, {json.RawMessage(calculatorDefinition), json.RawMessage(`false`)}} {
		f := &calculatorWire{t: t}
		out, r, e := calculatorClient(f).Subscriptions.Calculate(context.Background(), defs)
		if e == nil || out != nil || r != nil || f.calls != 0 {
			t.Fatal("invalid input dispatched")
		}
	}
}
func TestSubscriptionCalculateResponses(t *testing.T) {
	for _, text := range []string{"null", "[]", "{", ""} {
		b := &calculatorBody{Reader: bytes.NewBufferString(text)}
		f := &calculatorWire{t: t, want: calculatorRequest, code: 200, body: b}
		out, _, e := calculatorClient(f).Subscriptions.Calculate(context.Background(), []json.RawMessage{json.RawMessage(calculatorDefinition)})
		if e == nil || out != nil || f.calls != 1 || b.closed != 1 {
			t.Fatal(text, e)
		}
	}
	for _, code := range []int{403, 404, 409, 500} {
		b := &calculatorBody{Reader: bytes.NewBufferString(`[{"error_type":"synthetic","error_message":"refused","error_point":null}]`)}
		f := &calculatorWire{t: t, want: calculatorRequest, code: code, body: b}
		out, r, e := calculatorClient(f).Subscriptions.Calculate(context.Background(), []json.RawMessage{json.RawMessage(calculatorDefinition)})
		var er *ErrorResponse
		if !errors.As(e, &er) || out != nil || r == nil || r.StatusCode != code || f.calls != 1 || b.closed != 1 {
			t.Fatal(out, r, e)
		}
	}
}
func TestSubscriptionCalculateTransport(t *testing.T) {
	sentinel := errors.New("synthetic transport")
	f := &calculatorWire{t: t, want: calculatorRequest, err: sentinel}
	c := calculatorClient(f)
	out, r, e := c.Subscriptions.Calculate(context.Background(), []json.RawMessage{json.RawMessage(calculatorDefinition)})
	if out != nil || r != nil || !errors.Is(e, sentinel) || f.calls != 1 {
		t.Fatal(out, r, e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out, r, e = c.Subscriptions.Calculate(ctx, []json.RawMessage{json.RawMessage(calculatorDefinition)})
	if out != nil || r != nil || !errors.Is(e, context.Canceled) || f.calls != 2 {
		t.Fatal(out, r, e)
	}
}
