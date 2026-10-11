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

type consoleWire struct {
	t           *testing.T
	body        string
	code, calls int
	err         error
}

func (f *consoleWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/servers/e27eacc3-0f20-4c1f-9f7d-eac705bf2d02/action/?do=open_console" || q.URL.Fragment != "" {
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

const consoleID = "e27eacc3-0f20-4c1f-9f7d-eac705bf2d02"

func consoleClient(f *consoleWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func TestServerOpenConsolePublicACK(t *testing.T) {
	b, e := os.ReadFile("server_open_console_response.json")
	if e != nil {
		t.Fatal(e)
	}
	var want ServerConsoleAction
	if e = json.Unmarshal(b, &want); e != nil {
		t.Fatal(e)
	}
	f := &consoleWire{t: t, body: string(b), code: 202}
	v, r, e := consoleClient(f).Servers.OpenConsole(context.Background(), consoleID)
	if e != nil || r == nil || r.StatusCode != 202 || !reflect.DeepEqual(v, &want) || f.calls != 1 || v.ConsoleURL != "csconsole://direct.wdc.alpha3cloud.com:43835" {
		t.Fatal("ack", v, r, e)
	}
}
func TestServerOpenConsoleInvalidZeroDispatch(t *testing.T) {
	f := &consoleWire{t: t}
	c := consoleClient(f)
	for _, id := range []string{"", "..", "a/b", "a?b", "a#b", "%2f", "x\\y", "x\n", "é"} {
		v, r, e := c.Servers.OpenConsole(context.Background(), id)
		if v != nil || r != nil || e == nil || f.calls != 0 {
			t.Fatal("invalid", id, v, r, e)
		}
	}
}
func TestServerOpenConsoleErrorsAndDecode(t *testing.T) {
	f := &consoleWire{t: t}
	c := consoleClient(f)
	for _, code := range []int{403, 404, 500} {
		f.code = code
		f.body = `[{"error_message":"denied","error_point":"server","error_type":"synthetic"}]`
		n := f.calls
		v, r, e := c.Servers.OpenConsole(context.Background(), consoleID)
		var api *ErrorResponse
		if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || !reflect.DeepEqual(api.Errors, []Error{{Message: "denied", Point: "server", Type: "synthetic"}}) || f.calls != n+1 {
			t.Fatal("HTTP", v, r, e)
		}
	}
	f.code = 202
	for _, body := range []string{"", `{`, `[]`, `1`} {
		f.body = body
		n := f.calls
		v, r, e := c.Servers.OpenConsole(context.Background(), consoleID)
		if v != nil || r != nil || e == nil || f.calls != n+1 {
			t.Fatal("decode", v, r, e)
		}
	}
	// Existing typed decoder accepts empty/null objects and ignores unknown fields.
	for _, body := range []string{`{}`, `null`, `{"unknown":true}`} {
		f.body = body
		n := f.calls
		v, r, e := c.Servers.OpenConsole(context.Background(), consoleID)
		if e != nil || r == nil || !reflect.DeepEqual(v, &ServerConsoleAction{}) || f.calls != n+1 {
			t.Fatal("permissive decoder", v, r, e)
		}
	}
	sentinel := errors.New("console transport")
	f.err = sentinel
	n := f.calls
	if v, r, e := c.Servers.OpenConsole(context.Background(), consoleID); v != nil || r != nil || !errors.Is(e, sentinel) || f.calls != n+1 {
		t.Fatal("transport", v, r, e)
	}
	f.err = nil
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n = f.calls
	if v, r, e := c.Servers.OpenConsole(ctx, consoleID); v != nil || r != nil || !errors.Is(e, context.Canceled) || f.calls != n+1 {
		t.Fatal("cancel", v, r, e)
	}
}
