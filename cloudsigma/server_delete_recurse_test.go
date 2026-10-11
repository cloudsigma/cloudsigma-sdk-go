package cloudsigma

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

type serverDeleteWire struct {
	t           *testing.T
	query       string
	code, calls int
	body        string
	err         error
}

func (f *serverDeleteWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.ContentLength != 0 {
		f.t.Fatal("body length")
	}
	if q.Body != nil {
		b, e := io.ReadAll(q.Body)
		if e != nil || len(b) != 0 {
			f.t.Fatal("nonempty body")
		}
	}
	if q.Method != "DELETE" || q.URL.EscapedPath() != "/api/2.0/servers/id/" || q.URL.RawQuery != f.query || q.URL.Fragment != "" {
		f.t.Fatal("delete wire")
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.body)), Request: q}, nil
}
func TestServerDeleteRecursiveWire(t *testing.T) {
	for _, mode := range []ServerDeleteRecurse{"", ServerDeleteAllDrives, ServerDeleteDisks, ServerDeleteCDROMs} {
		name := string(mode)
		if name == "" {
			name = "default"
		}
		t.Run(name, func(t *testing.T) {
			query := ""
			if mode != "" {
				query = "recurse=" + string(mode)
			}
			f := &serverDeleteWire{t: t, query: query, code: 204, body: "not JSON: acknowledgement only"}
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
			var r *Response
			var e error
			if mode == "" {
				r, e = c.Servers.Delete(context.Background(), "id")
			} else {
				r, e = c.Servers.DeleteRecursive(context.Background(), "id", mode)
			}
			if e != nil || r == nil || r.StatusCode != 204 || f.calls != 1 {
				t.Fatal("ack")
			}
		})
	}
}
func TestServerDeleteRecursiveRefusals(t *testing.T) {
	f := &serverDeleteWire{t: t}
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	for _, id := range []string{"", "..", "a/b", `a\b`, "?x", "#x", "%2f", "%252f", "é", "\n"} {
		r, e := c.Servers.DeleteRecursive(context.Background(), id, ServerDeleteDisks)
		if r != nil || e == nil || f.calls != 0 {
			t.Fatal("identifier")
		}
		if id == "" && !errors.Is(e, ErrEmptyArgument) {
			t.Fatal("empty")
		}
	}
	for _, mode := range []ServerDeleteRecurse{"", "all", "disks&other=1"} {
		r, e := c.Servers.DeleteRecursive(context.Background(), "id", mode)
		if r != nil || e == nil || f.calls != 0 {
			t.Fatal("mode")
		}
	}
}
func TestServerDeleteRecursiveErrors(t *testing.T) {
	for _, method := range []string{"default", "recursive"} {
		t.Run(method, func(t *testing.T) {
			query := ""
			if method == "recursive" {
				query = "recurse=disks"
			}
			call := func(c *Client, ctx context.Context) (*Response, error) {
				if method == "default" {
					return c.Servers.Delete(ctx, "id")
				}
				return c.Servers.DeleteRecursive(ctx, "id", ServerDeleteDisks)
			}
			for _, code := range []int{403, 404, 500} {
				t.Run(strconv.Itoa(code), func(t *testing.T) {
					f := &serverDeleteWire{t: t, query: query, code: code, body: `[{"error_message":"denied","error_point":"uuid","error_type":"forbidden"}]`}
					c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
					r, e := call(c, context.Background())
					var api *ErrorResponse
					if r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0] != (Error{Message: "denied", Point: "uuid", Type: "forbidden"}) || f.calls != 1 {
						t.Fatal("error fidelity")
					}
				})
			}
			for _, name := range []string{"transport", "cancellation"} {
				t.Run(name, func(t *testing.T) {
					sentinel := errors.New("transport")
					ctx := context.Background()
					f := &serverDeleteWire{t: t, query: query, err: sentinel}
					if name == "cancellation" {
						var stop context.CancelFunc
						ctx, stop = context.WithCancel(ctx)
						stop()
						sentinel = context.Canceled
					}
					c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
					r, e := call(c, ctx)
					if r != nil || !errors.Is(e, sentinel) || f.calls != 1 {
						t.Fatal("transport/context")
					}
				})
			}
		})
	}
}
