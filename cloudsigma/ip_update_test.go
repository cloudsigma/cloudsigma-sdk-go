package cloudsigma

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type ipUpdateWire struct {
	t              *testing.T
	body, response string
	code, calls    int
	err            error
}

func (f *ipUpdateWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	b, e := io.ReadAll(q.Body)
	if e != nil || q.Method != "PUT" || q.URL.RequestURI() != "/api/2.0/ips/185.12.6.183/" || q.URL.Fragment != "" || !definitionJSONEqual(string(b), f.body) {
		f.t.Fatal("wire mismatch")
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.response)), Request: q}, nil
}
func TestIPUpdateRepresentation(t *testing.T) {
	for _, tc := range []struct {
		name string
		meta map[string]string
		wire string
	}{
		{"public", map[string]string{"custom_field": "some custom data", "name": "my ip"}, `{"meta":{"custom_field":"some custom data","name":"my ip"}}`},
		{"empty", map[string]string{}, `{"meta":{}}`},
		{"ascii32", map[string]string{strings.Repeat("a", 32): ""}, `{"meta":{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa":""}}`},
		{"unicode32", map[string]string{strings.Repeat("é", 32): "value"}, `{"meta":{"éééééééééééééééééééééééééééééééé":"value"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := map[string]string{}
			for k, v := range tc.meta {
				before[k] = v
			}
			f := &ipUpdateWire{t: t, body: tc.wire, code: 200, response: `{"uuid":"185.12.6.183","gateway":"185.12.6.1","netmask":24,"nameservers":["192.0.2.53"],"meta":{"custom_field":"some custom data","name":"my ip"},"owner":{"uuid":"owner","resource_uri":"/owner/"},"resource_uri":"/api/2.0/ips/185.12.6.183/","server":null}`}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			v, r, e := c.IPs.Update(context.Background(), "185.12.6.183", &IPUpdateRequest{Meta: tc.meta})
			want := IP{UUID: "185.12.6.183", Gateway: "185.12.6.1", Netmask: 24, Nameservers: []string{"192.0.2.53"}, Meta: map[string]interface{}{"custom_field": "some custom data", "name": "my ip"}, Owner: &ResourceLink{UUID: "owner", ResourceURI: "/owner/"}, ResourceURI: "/api/2.0/ips/185.12.6.183/"}
			if e != nil || r == nil || r.StatusCode != 200 || v == nil || !reflect.DeepEqual(*v, want) || f.calls != 1 {
				t.Fatal("modeled response")
			}
			if !reflect.DeepEqual(before, tc.meta) {
				t.Fatal("map mutation")
			}
		})
	}
}
func TestIPUpdateRefusals(t *testing.T) {
	f := &ipUpdateWire{t: t}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	d := &IPUpdateRequest{Meta: map[string]string{}}
	for _, id := range []string{"", "id", "185.12.6.183/x", "185.12.6.183?x", "185.12.6.183#x", `185.12.6.183\x`, "%31.2.3.4", "1.2.3.4/24", "::1", "::ffff:185.12.6.183", "fe80::1%eth0", "01.2.3.4", "1.2.3.999", " 1.2.3.4", "1.2.3.4\n"} {
		v, r, e := c.IPs.Update(context.Background(), id, d)
		if v != nil || r != nil || e == nil || f.calls != 0 {
			t.Fatal("identifier refusal")
		}
		if id == "" && !errors.Is(e, ErrEmptyArgument) {
			t.Fatal("empty error")
		}
	}
	for _, body := range []*IPUpdateRequest{nil, {}, {Meta: map[string]string{"good": "retained", strings.Repeat("a", 33): "bad"}}, {Meta: map[string]string{strings.Repeat("é", 33): "bad"}}, {Meta: map[string]string{string([]byte{255}): "bad"}}, {Meta: map[string]string{"good": string([]byte{255})}}} {
		v, r, e := c.IPs.Update(context.Background(), "185.12.6.183", body)
		if v != nil || r != nil || e == nil || f.calls != 0 {
			t.Fatal("payload refusal")
		}
	}
}
func TestIPUpdateErrors(t *testing.T) {
	d := &IPUpdateRequest{Meta: map[string]string{}}
	for _, code := range []int{403, 404, 500} {
		f := &ipUpdateWire{t: t, body: `{"meta":{}}`, code: code, response: `[{"error_message":"denied","error_point":"meta","error_type":"forbidden"}]`}
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
		v, r, e := c.IPs.Update(context.Background(), "185.12.6.183", d)
		var api *ErrorResponse
		if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0] != (Error{Message: "denied", Point: "meta", Type: "forbidden"}) || f.calls != 1 {
			t.Fatal("API error")
		}
	}
	for _, cancel := range []bool{false, true} {
		sentinel := errors.New("transport")
		f := &ipUpdateWire{t: t, body: `{"meta":{}}`, err: sentinel}
		ctx := context.Background()
		if cancel {
			var stop context.CancelFunc
			ctx, stop = context.WithCancel(ctx)
			stop()
			sentinel = context.Canceled
		}
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
		v, r, e := c.IPs.Update(ctx, "185.12.6.183", d)
		if v != nil || r != nil || !errors.Is(e, sentinel) || f.calls != 1 {
			t.Fatal("transport/context")
		}
	}
	for _, body := range []string{``, `{`, `[]`, `1`, `{"netmask":"bad"}`, `null`, `{}`} {
		f := &ipUpdateWire{t: t, body: `{"meta":{}}`, response: body, code: 200}
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
		v, r, e := c.IPs.Update(context.Background(), "185.12.6.183", d)
		if body == `null` || body == `{}` {
			if e != nil || r == nil || v == nil || !reflect.DeepEqual(*v, IP{}) {
				t.Fatal("permissive singleton")
			}
		} else {
			if e == nil || v != nil || r != nil {
				t.Fatal("decode failure")
			}
			if body == `` && !errors.Is(e, io.EOF) {
				t.Fatal("EOF")
			}
		}
		if f.calls != 1 {
			t.Fatal("count")
		}
	}
}
