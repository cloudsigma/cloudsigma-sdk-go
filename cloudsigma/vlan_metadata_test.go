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

type vlanMetadataWire struct {
	t              *testing.T
	body, response string
	code, calls    int
	err            error
}

func (f *vlanMetadataWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	b, e := io.ReadAll(q.Body)
	if e != nil || q.Method != "PUT" || q.URL.RequestURI() != "/api/2.0/vlans/96537817-f4b6-496b-a861-e74192d3ccb0/" || q.URL.Fragment != "" || !definitionJSONEqual(string(b), f.body) {
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
func TestVLANMetadataRepresentation(t *testing.T) {
	for _, tc := range []struct {
		name string
		meta map[string]string
		wire string
	}{
		{"public", map[string]string{"custom_field": "some custom data", "name": "my vlan"}, `{"meta":{"custom_field":"some custom data","name":"my vlan"}}`},
		{"empty", map[string]string{}, `{"meta":{}}`},
		{"ascii32", map[string]string{strings.Repeat("a", 32): ""}, `{"meta":{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa":""}}`},
		{"unicode32", map[string]string{strings.Repeat("é", 32): "value"}, `{"meta":{"éééééééééééééééééééééééééééééééé":"value"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := map[string]string{}
			for k, v := range tc.meta {
				before[k] = v
			}
			f := &vlanMetadataWire{t: t, body: tc.wire, code: 200, response: `{"uuid":"96537817-f4b6-496b-a861-e74192d3ccb0","meta":{"custom_field":"some custom data","name":"my vlan"},"owner":{"uuid":"owner","resource_uri":"/owner/"},"resource_uri":"/vlans/result/","servers":[],"tags":[],"subscription":{"id":12,"resource_uri":"/subscription/12/"},"bottom_vlan_tag":null,"grantees":[],"permissions":[]}`}

			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			v, r, e := c.VLANs.UpdateMetadata(context.Background(), "96537817-f4b6-496b-a861-e74192d3ccb0", &VLANMetadataUpdateRequest{Meta: tc.meta})
			want := VLAN{UUID: "96537817-f4b6-496b-a861-e74192d3ccb0", Meta: map[string]interface{}{"custom_field": "some custom data", "name": "my vlan"}, Owner: &ResourceLink{UUID: "owner", ResourceURI: "/owner/"}, ResourceURI: "/vlans/result/", Servers: []ResourceLink{}, Tags: []Tag{}, Subscription: &VLANSubscription{ID: 12, ResourceURI: "/subscription/12/"}}

			if e != nil || r == nil || r.StatusCode != 200 || v == nil || !reflect.DeepEqual(*v, want) || f.calls != 1 {
				t.Fatal("modeled response")
			}
			if !reflect.DeepEqual(before, tc.meta) {
				t.Fatal("map mutation")
			}
		})
	}
}
func TestVLANMetadataRefusals(t *testing.T) {
	f := &vlanMetadataWire{t: t}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	d := &VLANMetadataUpdateRequest{Meta: map[string]string{}}
	for _, id := range []string{"", ".", "..", "a/b", `a\b`, "a?x", "a#x", "%2f", "%252f", "é", "a\n", " a", "a.b", "a:b"} {
		v, r, e := c.VLANs.UpdateMetadata(context.Background(), id, d)
		if v != nil || r != nil || e == nil || f.calls != 0 {
			t.Fatal("identifier refusal")
		}
		if id == "" && !errors.Is(e, ErrEmptyArgument) {
			t.Fatal("empty error")
		}
	}
	for _, body := range []*VLANMetadataUpdateRequest{nil, {}, {Meta: map[string]string{"good": "retained", strings.Repeat("a", 33): "bad"}}, {Meta: map[string]string{strings.Repeat("é", 33): "bad"}}, {Meta: map[string]string{string([]byte{255}): "bad"}}, {Meta: map[string]string{"good": string([]byte{255})}}} {
		v, r, e := c.VLANs.UpdateMetadata(context.Background(), "96537817-f4b6-496b-a861-e74192d3ccb0", body)
		if v != nil || r != nil || e == nil || f.calls != 0 {
			t.Fatal("payload refusal")
		}
	}
}
func TestVLANMetadataErrors(t *testing.T) {
	d := &VLANMetadataUpdateRequest{Meta: map[string]string{}}
	for _, code := range []int{403, 404, 500} {
		f := &vlanMetadataWire{t: t, body: `{"meta":{}}`, code: code, response: `[{"error_message":"denied","error_point":"meta","error_type":"forbidden"}]`}
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
		v, r, e := c.VLANs.UpdateMetadata(context.Background(), "96537817-f4b6-496b-a861-e74192d3ccb0", d)
		var api *ErrorResponse
		if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0] != (Error{Message: "denied", Point: "meta", Type: "forbidden"}) || f.calls != 1 {
			t.Fatal("API error")
		}
	}
	for _, cancel := range []bool{false, true} {
		sentinel := errors.New("transport")
		f := &vlanMetadataWire{t: t, body: `{"meta":{}}`, err: sentinel}
		ctx := context.Background()
		if cancel {
			var stop context.CancelFunc
			ctx, stop = context.WithCancel(ctx)
			stop()
			sentinel = context.Canceled
		}
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
		v, r, e := c.VLANs.UpdateMetadata(ctx, "96537817-f4b6-496b-a861-e74192d3ccb0", d)
		if v != nil || r != nil || !errors.Is(e, sentinel) || f.calls != 1 {
			t.Fatal("transport/context")
		}
	}
	for _, body := range []string{``, `{`, `[]`, `1`, `{"subscription":{"id":"bad"}}`, `null`, `{}`} {
		f := &vlanMetadataWire{t: t, body: `{"meta":{}}`, response: body, code: 200}
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
		v, r, e := c.VLANs.UpdateMetadata(context.Background(), "96537817-f4b6-496b-a861-e74192d3ccb0", d)
		if body == `null` || body == `{}` {
			if e != nil || r == nil || v == nil || !reflect.DeepEqual(*v, VLAN{}) {
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
