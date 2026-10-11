package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"testing"
)

type ipRawWire struct {
	t           *testing.T
	data        []byte
	code, calls int
	err         error
}

func (f *ipRawWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/ips/185.12.6.183/" || q.URL.Fragment != "" || q.ContentLength != 0 {
		f.t.Fatal("route")
	}
	if q.Body != nil {
		b, e := io.ReadAll(q.Body)
		if e != nil || len(b) != 0 {
			f.t.Fatal("body")
		}
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(f.data)), Request: q}, nil
}
func TestIPRawPublic(t *testing.T) {
	data, e := os.ReadFile("ip_raw_public.json")
	if e != nil {
		t.Fatal(e)
	}
	var expected map[string]json.RawMessage
	if json.Unmarshal(data, &expected) != nil {
		t.Fatal("fixture")
	}
	f := &ipRawWire{t: t, data: data, code: 200}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	v, r, e := c.IPs.GetRaw(context.Background(), "185.12.6.183")
	if e != nil || r == nil || r.StatusCode != 200 || !reflect.DeepEqual(v, expected) || f.calls != 1 {
		t.Fatal("public preservation")
	}
	for k, w := range map[string]string{"grantees": "[]", "permissions": "[]", "tags": "[]"} {
		if string(v[k]) != w {
			t.Fatal("field")
		}
	}
	if !definitionJSONEqual(string(v["subscription"]), `{"id":7273,"resource_uri":"/api/2.0/subscriptions/7273/"}`) {
		t.Fatal("subscription")
	}
	snapshot := append([]byte(nil), v["uuid"]...)
	pos := bytes.Index(data, []byte(`"uuid": "185.12.6.183"`))
	if pos < 0 {
		t.Fatal("fixture UUID range")
	}
	data[pos+9] = 'X'
	if !bytes.Equal(v["uuid"], snapshot) {
		t.Fatal("input alias")
	}
	// Restore input before snapshotting; never erase evidence after result mutation.
	data[pos+9] = '1'
	inputSnapshot := append([]byte(nil), data...)
	v["uuid"][1] = 'X'
	if string(v["uuid"]) != `"X85.12.6.183"` {
		t.Fatal("result mutation did not occur")
	}
	if !bytes.Equal(data, inputSnapshot) {
		t.Fatal("result alias")
	}
}
func TestIPRawSynthetic(t *testing.T) {
	f := &ipRawWire{t: t, code: 200, data: []byte(`{"synthetic_zero":0,"grantees":[{"unknown":9007199254740993}],"permissions":[],"false":false,"empty":"","null":null,"next":"https://evil.invalid/"}`)}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	v, r, e := c.IPs.GetRaw(context.Background(), "185.12.6.183")
	want := map[string]json.RawMessage{"synthetic_zero": json.RawMessage(`0`), "grantees": json.RawMessage(`[{"unknown":9007199254740993}]`), "permissions": json.RawMessage(`[]`), "false": json.RawMessage(`false`), "empty": json.RawMessage(`""`), "null": json.RawMessage(`null`), "next": json.RawMessage(`"https://evil.invalid/"`)}
	if e != nil || r == nil || !reflect.DeepEqual(v, want) || f.calls != 1 {
		t.Fatal("synthetic precision/no traversal")
	}
	again, _, e := c.IPs.GetRaw(context.Background(), "185.12.6.183")
	v["synthetic_zero"][0] = '9'
	if e != nil || string(again["synthetic_zero"]) != "0" || f.calls != 2 {
		t.Fatal("result independence")
	}
}
func TestIPRawBoundaries(t *testing.T) {
	f := &ipRawWire{t: t, code: 200}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	for _, id := range []string{"", "..", "a/b", "a?b", "a#b", `a\b`, "%2f", "é", "\n", "::1", "::ffff:185.12.6.183", "fe80::1%eth0", "1.2.3.4/24", "01.2.3.4", "1.2.3.999"} {
		v, r, e := c.IPs.GetRaw(context.Background(), id)
		if v != nil || r != nil || e == nil || f.calls != 0 {
			t.Fatal("refusal")
		}
	}
	for _, body := range []string{``, `{`, `[]`, `1`, `null`, `{}`} {
		f.calls = 0
		f.data = []byte(body)
		v, r, e := c.IPs.GetRaw(context.Background(), "185.12.6.183")
		if body == `{}` {
			if e != nil || v == nil || len(v) != 0 || r == nil {
				t.Fatal("empty object")
			}
		} else {
			if e == nil || v != nil {
				t.Fatal("decode")
			}
			if body == `null` {
				if r == nil {
					t.Fatal("null response")
				}
			} else if r != nil {
				t.Fatal("decode response")
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
		f.code = code
		f.calls = 0
		f.data = []byte(`[{"error_message":"denied","error_point":"uuid","error_type":"forbidden"}]`)
		v, r, e := c.IPs.GetRaw(context.Background(), "185.12.6.183")
		var api *ErrorResponse
		if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0] != (Error{Message: "denied", Point: "uuid", Type: "forbidden"}) || f.calls != 1 {
			t.Fatal("API error")
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
		v, r, e := c.IPs.GetRaw(ctx, "185.12.6.183")
		if v != nil || r != nil || !errors.Is(e, sentinel) || f.calls != 1 {
			t.Fatal("transport/context")
		}
	}
}

// Negative control: the same whole-input oracle must detect shared storage.
// This fixture does not use or replace production decoding.
func TestIPRawOwnershipOracle(t *testing.T) {
	for _, shared := range []bool{false, true} {
		input := []byte(`{"uuid":"185.12.6.183"}`)
		start := bytes.Index(input, []byte(`"185.12.6.183"`))
		result := input[start : start+len(`"185.12.6.183"`)]
		if !shared {
			result = append([]byte(nil), result...)
		}
		snapshot := append([]byte(nil), input...)
		result[1] = 'X'
		if string(result) != `"X85.12.6.183"` {
			t.Fatal("control mutation")
		}
		unchanged := bytes.Equal(input, snapshot)
		if unchanged == shared {
			t.Fatal("ownership oracle failed control")
		}
	}
}
