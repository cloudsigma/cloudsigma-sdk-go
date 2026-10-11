package cloudsigma

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"
)

func definitionCall(c *Client, ctx context.Context, create bool, ds []ServerDefinition) ([]Server, *Response, error) {
	if create {
		return c.Servers.CreateDefinition(ctx, ds)
	}
	v, r, e := c.Servers.UpdateDefinition(ctx, "id", ds[0])
	if v == nil {
		return nil, r, e
	}
	return []Server{*v}, r, e
}
func serverWriteDefinitionWire(t *testing.T, create bool, body, response string, code int) *defWire {
	f := &defWire{t: t, method: "PUT", path: "/api/2.0/servers/id/", body: body, response: response, code: code}
	if create {
		f.method = "POST"
		f.path = "/api/2.0/servers/"
	}
	return f
}
func TestDefinitionV2Responses(t *testing.T) {
	for _, create := range []bool{true, false} {
		d := ServerDefinition{"name": json.RawMessage(`""`), "a": json.RawMessage(`[]`), "o": json.RawMessage(`{}`), "nested": json.RawMessage(`{"uuid":null}`)}
		ds := []ServerDefinition{d}
		body := `{"name":"","a":[],"o":{},"nested":{"uuid":null}}`
		one := `{"uuid":"first","jobs":[{"n":9007199254740993}],"drives":[{"drive":{"licenses":[{"license":{"free":9007199254740993,"initial_purchase_requirement":0,"subsequent_purchase_requirement":2.5}}]}}]}`
		response := one
		code := 200
		count := 1
		if create {
			ds = append(ds, ServerDefinition{"name": json.RawMessage(`"second"`)})
			body = `{"objects":[` + body + `,{"name":"second"}]}`
			response = `{"objects":[` + one + `,{"uuid":"second","jobs":null}]}`
			code = 201
			count = 2
		}
		before, _ := json.Marshal(ds)
		f := serverWriteDefinitionWire(t, create, body, response, code)
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
		v, r, e := definitionCall(c, context.Background(), create, ds)
		after, _ := json.Marshal(ds)
		if e != nil || r == nil || r.StatusCode != code || len(v) != count || f.calls != 1 || !reflect.DeepEqual(before, after) {
			t.Fatal("response/count/mutation")
		}
		if v[0].UUID != "first" || !definitionJSONEqual(string(v[0].Jobs), `[{"n":9007199254740993}]`) {
			t.Fatal("modeled response")
		}
		l := v[0].Drives[0].Drive.Licenses[0].License
		if l.Free != "9007199254740993" || l.InitialPurchaseRequirement != "0" || l.SubsequentPurchaseRequirement != "2.5" {
			t.Fatal("license")
		}
		if create && v[1].UUID != "second" {
			t.Fatal("order")
		}
	}
}
func TestDefinitionV2Refusals(t *testing.T) {
	bad := []ServerDefinition{nil, {"x": nil}, {"x": json.RawMessage(``)}, {"x": json.RawMessage(`{`)}}
	for _, k := range []string{"jobs", "owner", "resource_uri", "runtime", "status", "uuid"} {
		bad = append(bad, ServerDefinition{k: json.RawMessage(`null`)})
	}
	for _, create := range []bool{true, false} {
		for _, d := range bad {
			f := &defWire{t: t}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			ds := []ServerDefinition{d}
			if create {
				ds = append([]ServerDefinition{{}}, ds...)
			}
			v, r, e := definitionCall(c, context.Background(), create, ds)
			if v != nil || r != nil || e == nil || f.calls != 0 {
				t.Fatal("refusal")
			}
		}
	}
	f := &defWire{t: t}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	v, r, e := c.Servers.CreateDefinition(context.Background(), nil)
	if v != nil || r != nil || !errors.Is(e, ErrEmptyPayloadNotAllowed) || f.calls != 0 {
		t.Fatal("nil batch")
	}
	for _, id := range []string{"", "?x", "#x", "a/b", `a\b`, "%2f", "%252f", ".", "..", "a.b", "\n", "é"} {
		v, r, e := c.Servers.UpdateDefinition(context.Background(), id, ServerDefinition{})
		if v != nil || r != nil || e == nil || f.calls != 0 {
			t.Fatal("id")
		}
		if id == "" && !errors.Is(e, ErrEmptyArgument) {
			t.Fatal("empty id")
		}
	}
}
func TestDefinitionV2Errors(t *testing.T) {
	for _, create := range []bool{true, false} {
		body := `{}`
		if create {
			body = `{"objects":[{}]}`
		}
		for _, code := range []int{403, 404, 500} {
			f := serverWriteDefinitionWire(t, create, body, `[{"error_message":"denied","error_point":"name","error_type":"forbidden"}]`, code)
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			v, r, e := definitionCall(c, context.Background(), create, []ServerDefinition{{}})
			var api *ErrorResponse
			if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0] != (Error{Message: "denied", Point: "name", Type: "forbidden"}) || f.calls != 1 {
				t.Fatal("API fidelity")
			}
		}
		for _, cancel := range []bool{false, true} {
			sentinel := errors.New("transport")
			f := serverWriteDefinitionWire(t, create, body, `{}`, 200)
			ctx := context.Background()
			if cancel {
				var stop context.CancelFunc
				ctx, stop = context.WithCancel(ctx)
				stop()
				sentinel = context.Canceled
			} else {
				f.err = sentinel
			}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			v, r, e := definitionCall(c, ctx, create, []ServerDefinition{{}})
			if v != nil || r != nil || !errors.Is(e, sentinel) || f.calls != 1 {
				t.Fatal("transport/context")
			}
		}
	}
}
func TestDefinitionV2DecodeBoundaries(t *testing.T) {
	for _, create := range []bool{true, false} {
		body := `{}`
		if create {
			body = `{"objects":[{}]}`
		}
		for _, response := range []string{`{`, ``, `[]`, `1`, `null`, `{}`, `{"objects":[]}`, `{"objects":null}`} {
			f := serverWriteDefinitionWire(t, create, body, response, 200)
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			v, r, e := definitionCall(c, context.Background(), create, []ServerDefinition{{}})
			wantError := response == `{` || response == `` || response == `[]` || response == `1`
			if f.calls != 1 {
				t.Fatal("dispatch count")
			}
			if wantError {
				if e == nil || v != nil || r != nil {
					t.Fatal("decode error return contract")
				}
				switch response {
				case ``:
					if !errors.Is(e, io.EOF) {
						t.Fatal("expected EOF")
					}
				case `{`:
					if !errors.Is(e, io.ErrUnexpectedEOF) {
						t.Fatal("expected unexpected EOF")
					}
				default:
					var mismatch *json.UnmarshalTypeError
					if !errors.As(e, &mismatch) {
						t.Fatal("expected type error")
					}
					want := "array"
					if response == `1` {
						want = "number"
					}
					if mismatch.Value != want {
						t.Fatal("type error value")
					}
				}
			} else {
				if e != nil || r == nil || r.StatusCode != 200 {
					t.Fatal("successful decode contract")
				}
				if create {
					if response == `{"objects":[]}` {
						if v == nil || len(v) != 0 {
							t.Fatal("empty collection")
						}
					} else if v != nil {
						t.Fatal("absent/null collection")
					}
				} else if len(v) != 1 || !reflect.DeepEqual(v[0], Server{}) {
					t.Fatal("permissive zero singleton")
				}
			}
		}
	}
	f := serverWriteDefinitionWire(t, true, `{"objects":[]}`, `{"objects":[]}`, 201)
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	v, r, e := c.Servers.CreateDefinition(context.Background(), []ServerDefinition{})
	if e != nil || r == nil || v == nil || len(v) != 0 || f.calls != 1 {
		t.Fatal("empty SDK policy")
	}
}
