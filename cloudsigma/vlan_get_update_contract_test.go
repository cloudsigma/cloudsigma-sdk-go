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

type vlanContractWire struct {
	t                      *testing.T
	method, body, response string
	code, calls            int
	err                    error
}

func (f *vlanContractWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != f.method || q.URL.RequestURI() != "/api/2.0/vlans/96537817-f4b6-496b-a861-e74192d3ccb0/" || q.URL.Fragment != "" {
		f.t.Fatal("route")
	}
	var b []byte
	var e error
	if q.Body != nil {
		b, e = io.ReadAll(q.Body)
	}
	if e != nil {
		f.t.Fatal("read")
	}
	if f.method == "GET" {
		if len(b) != 0 || q.ContentLength != 0 {
			f.t.Fatal("GET body")
		}
	} else if !definitionJSONEqual(string(b), f.body) {
		f.t.Fatal("PUT body")
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.response)), Request: q}, nil
}
func TestVLANGetUpdateContract(t *testing.T) {
	for _, method := range []string{"GET", "PUT"} {
		t.Run(method, func(t *testing.T) {
			input := &VLANUpdateRequest{VLAN: &VLAN{Meta: map[string]interface{}{"custom_field": "some custom data", "name": "my vlan"}}}
			before, _ := json.Marshal(input)
			f := &vlanContractWire{t: t, method: method, body: `{"meta":{"custom_field":"some custom data","name":"my vlan"}}`, code: 200, response: `{"bottom_vlan_tag":null,"grantees":[],"permissions":[],"meta":{"custom_field":"some custom data","name":"my vlan"},"owner":{"uuid":"owner","resource_uri":"/owner/"},"resource_uri":"/vlans/result/","servers":[],"subscription":{"id":12,"resource_uri":"/subscriptions/12/"},"tags":[],"uuid":"result"}`}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			var v *VLAN
			var r *Response
			var e error
			if method == "GET" {
				v, r, e = c.VLANs.Get(context.Background(), "96537817-f4b6-496b-a861-e74192d3ccb0")
			} else {
				v, r, e = c.VLANs.Update(context.Background(), "96537817-f4b6-496b-a861-e74192d3ccb0", input)
			}
			want := VLAN{Meta: map[string]interface{}{"custom_field": "some custom data", "name": "my vlan"}, Owner: &ResourceLink{UUID: "owner", ResourceURI: "/owner/"}, ResourceURI: "/vlans/result/", Servers: []ResourceLink{}, Subscription: &VLANSubscription{ID: 12, ResourceURI: "/subscriptions/12/"}, Tags: []Tag{}, UUID: "result"}
			after, _ := json.Marshal(input)
			if e != nil || r == nil || r.StatusCode != 200 || v == nil || !reflect.DeepEqual(*v, want) || f.calls != 1 || string(before) != string(after) {
				t.Fatal("modeled response/nonmutation/count")
			}
		})
	}
}
func TestVLANGetUpdateBoundaries(t *testing.T) {
	for _, method := range []string{"GET", "PUT"} {
		t.Run(method, func(t *testing.T) {
			f := &vlanContractWire{t: t, method: method, body: `{}`, code: 200}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			call := func(ctx context.Context, id string) (*VLAN, *Response, error) {
				if method == "GET" {
					return c.VLANs.Get(ctx, id)
				}
				// Current empty-Meta encoding is {}: it cannot express documented
				// empty full replacement. Nil embedded VLAN also encodes {} and
				// response-field echo is not authorized by the metadata-only contract.
				// Mocked 200 is not server acceptance; intentional fixes must revise
				// or supersede these characterization expectations, not preserve gaps.
				return c.VLANs.Update(ctx, id, &VLANUpdateRequest{VLAN: &VLAN{Meta: map[string]interface{}{}}})
			}
			for _, id := range []string{"", "%"} {
				v, r, e := call(context.Background(), id)
				if v != nil || r != nil || e == nil || f.calls != 0 {
					t.Fatal("legacy refusal")
				}
				if id == "" && !errors.Is(e, ErrEmptyArgument) {
					t.Fatal("empty")
				}
			}
			id := "96537817-f4b6-496b-a861-e74192d3ccb0"
			for _, body := range []string{``, `{`, `[]`, `1`, `{"subscription":{"id":"bad"}}`, `null`, `{}`} {
				f.calls = 0
				f.response = body
				v, r, e := call(context.Background(), id)
				if body == `null` || body == `{}` {
					if e != nil || r == nil || v == nil || !reflect.DeepEqual(*v, VLAN{}) {
						t.Fatal("permissive")
					}
				} else {
					if e == nil || v != nil || r != nil {
						t.Fatal("decode")
					}
					if body == `` && !errors.Is(e, io.EOF) {
						t.Fatal("EOF")
					}
				}
				if f.calls != 1 {
					t.Fatal("count")
				}
			}
			for _, code := range []int{403, 404, 500} {
				f.calls = 0
				f.code = code
				f.response = `[{"error_message":"denied","error_point":"uuid","error_type":"forbidden"}]`
				v, r, e := call(context.Background(), id)
				var api *ErrorResponse
				if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0] != (Error{Message: "denied", Point: "uuid", Type: "forbidden"}) || f.calls != 1 {
					t.Fatal("typed error")
				}
			}
			for _, cancel := range []bool{false, true} {
				f.calls = 0
				sentinel := errors.New("transport")
				f.err = sentinel
				ctx := context.Background()
				if cancel {
					var stop context.CancelFunc
					ctx, stop = context.WithCancel(ctx)
					stop()
					sentinel = context.Canceled
				}
				v, r, e := call(ctx, id)
				if v != nil || r != nil || !errors.Is(e, sentinel) || f.calls != 1 {
					t.Fatal("transport/context")
				}
			}
		})
	}
}
// Legacy characterization only, not normative API behavior: empty Meta encodes
// {}, so it cannot express documented empty full metadata replacement; a nil
// embedded VLAN also encodes {}. Response-field echo is NOT authorized by the
// public metadata-only edit contract. Mocked 200 is not server acceptance.
// Future intentional fixes must revise/supersede these expectations rather than
// preserve these gaps as contract invariants.
func TestVLANUpdateLegacySerialization(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request *VLANUpdateRequest
		wire    string
	}{{"emptyMeta", &VLANUpdateRequest{VLAN: &VLAN{Meta: map[string]interface{}{}}}, `{}`}, {"nilEmbedded", &VLANUpdateRequest{}, `{}`}, {"responseFieldsEcho", &VLANUpdateRequest{VLAN: &VLAN{UUID: "echo", ResourceURI: "/echo/"}}, `{"uuid":"echo","resource_uri":"/echo/"}`}} {
		t.Run(tc.name, func(t *testing.T) {
			f := &vlanContractWire{t: t, method: "PUT", body: tc.wire, code: 200, response: `{}`}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			v, r, e := c.VLANs.Update(context.Background(), "96537817-f4b6-496b-a861-e74192d3ccb0", tc.request)
			if e != nil || r == nil || v == nil || f.calls != 1 {
				t.Fatal("legacy characterization")
			}
		})
	}
	f := &vlanContractWire{t: t}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	v, r, e := c.VLANs.Update(context.Background(), "id", nil)
	if v != nil || r != nil || !errors.Is(e, ErrEmptyPayloadNotAllowed) || f.calls != 0 {
		t.Fatal("nil request")
	}
}
