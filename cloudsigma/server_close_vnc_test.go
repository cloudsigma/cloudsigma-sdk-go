package cloudsigma

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
)

type closeVNCWire struct {
	t           *testing.T
	body        string
	code, calls int
	err         error
}

func (f *closeVNCWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/servers/42cb8991-a39a-4ed3-b549-efd6a1dedd48/action/?do=close_vnc" || q.URL.Fragment != "" {
		f.t.Fatal("route", q.URL)
	}
	b, e := io.ReadAll(q.Body)
	if e != nil || strings.TrimSpace(string(b)) != "{}" {
		f.t.Fatal("body", string(b), e)
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.body)), Request: q}, nil
}

const closeVNCID = "42cb8991-a39a-4ed3-b549-efd6a1dedd48"

func closeVNCClient(f *closeVNCWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func TestServerCloseVNCPublicACK(t *testing.T) {
	b, e := os.ReadFile("server_close_vnc_response.json")
	if e != nil {
		t.Fatal(e)
	}
	var want ServerAction
	if e = json.Unmarshal(b, &want); e != nil {
		t.Fatal(e)
	}
	f := &closeVNCWire{t: t, body: string(b), code: 202}
	v, r, e := closeVNCClient(f).Servers.CloseVNC(context.Background(), closeVNCID)
	if e != nil || r == nil || r.StatusCode != 202 || !reflect.DeepEqual(v, &want) || f.calls != 1 {
		t.Fatal("ack", v, r, e)
	}
}
func TestServerCloseVNCInvalidZeroDispatch(t *testing.T) {
	f := &closeVNCWire{t: t}
	c := closeVNCClient(f)
	for _, id := range []string{"", "..", "a/b", "a?b", "a#b", "%2f", "x\\y", "x\n", "é"} {
		v, r, e := c.Servers.CloseVNC(context.Background(), id)
		if v != nil || r != nil || e == nil || f.calls != 0 {
			t.Fatal("invalid", id, v, r, e)
		}
	}
}
func TestServerCloseVNCErrorsAndDecode(t *testing.T) {
	f := &closeVNCWire{t: t}
	c := closeVNCClient(f)
	for _, code := range []int{403, 404, 500} {
		f.code = code
		f.body = `[{"error_message":"denied","error_point":"server","error_type":"synthetic"}]`
		n := f.calls
		v, r, e := c.Servers.CloseVNC(context.Background(), closeVNCID)
		var api *ErrorResponse
		if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || !reflect.DeepEqual(api.Errors, []Error{{Message: "denied", Point: "server", Type: "synthetic"}}) || f.calls != n+1 {
			t.Fatal("HTTP", v, r, e)
		}
	}
	f.code = 202
	for _, body := range []string{"", `{`, `[]`, `1`} {
		f.body = body
		n := f.calls
		v, r, e := c.Servers.CloseVNC(context.Background(), closeVNCID)
		if v != nil || r != nil || e == nil || f.calls != n+1 {
			t.Fatal("decode", v, r, e)
		}
	}
	// Existing typed decoder accepts empty/null objects and ignores unknown fields.
	for _, body := range []string{`{}`, `null`, `{"unknown":true}`} {
		f.body = body
		n := f.calls
		v, r, e := c.Servers.CloseVNC(context.Background(), closeVNCID)
		if e != nil || r == nil || !reflect.DeepEqual(v, &ServerAction{}) || f.calls != n+1 {
			t.Fatal("permissive decoder", v, r, e)
		}
	}
	sentinel := errors.New("console transport")
	f.err = sentinel
	n := f.calls
	if v, r, e := c.Servers.CloseVNC(context.Background(), closeVNCID); v != nil || r != nil || !errors.Is(e, sentinel) || f.calls != n+1 {
		t.Fatal("transport", v, r, e)
	}
	f.err = nil
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n = f.calls
	if v, r, e := c.Servers.CloseVNC(ctx, closeVNCID); v != nil || r != nil || !errors.Is(e, context.Canceled) || f.calls != n+1 {
		t.Fatal("cancel", v, r, e)
	}
}
