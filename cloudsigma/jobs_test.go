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

type jobWire struct {
	t                   *testing.T
	calls, closed, code int
	body                string
	err                 error
}

func (f *jobWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "GET" || q.URL.Path != "/api/2.0/jobs/4e1944be-a078-4ead-9d19-5f8823a84d42/" || q.URL.RawQuery != "" || q.URL.Fragment != "" {
		f.t.Fatal("route", q.URL)
	}
	if q.Body != nil {
		raw, e := io.ReadAll(q.Body)
		if e != nil || len(raw) != 0 {
			f.t.Fatal("body", string(raw), e)
		}
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: &jobBody{strings.NewReader(f.body), f}, Request: q}, nil
}

type jobBody struct {
	*strings.Reader
	owner *jobWire
}

func (b *jobBody) Close() error { b.owner.closed++; return nil }
func jobClient(f *jobWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}

const jobTestID = "4e1944be-a078-4ead-9d19-5f8823a84d42"

func TestJobsGetPublicResponse(t *testing.T) {
	raw, e := os.ReadFile("job_get_response.json")
	if e != nil {
		t.Fatal(e)
	}
	f := &jobWire{t: t, code: 200, body: string(raw)}
	c := jobClient(f)
	got, r, e := c.Jobs.Get(context.Background(), jobTestID)
	var want map[string]json.RawMessage
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if e != nil || r == nil || r.StatusCode != 200 || !reflect.DeepEqual(got, want) || f.calls != 1 || f.closed != 1 {
		t.Fatal(got, r, e)
	}
	var children []map[string]json.RawMessage
	if e = json.Unmarshal(got["children"], &children); e != nil || len(children) != 1 || string(children[0]["operation"]) != `"drive_clone"` || string(got["uuid"]) != `"`+jobTestID+`"` {
		t.Fatal(children, e)
	}
	// Witness the mutation before checking that the independently decoded second
	// result and fixture-derived expectation did not share its raw storage.
	second, _, e := c.Jobs.Get(context.Background(), jobTestID)
	if e != nil {
		t.Fatal(e)
	}
	got["uuid"][1] = 'X'
	if got["uuid"][1] != 'X' {
		t.Fatal("mutation did not occur")
	}
	if !reflect.DeepEqual(second, want) || f.calls != 2 || f.closed != 2 {
		t.Fatal("aliased results")
	}
}
func TestJobsGetPresenceAndInvalidIDs(t *testing.T) {
	f := &jobWire{t: t, code: 200, body: `{"data":{"progress":0,"large":9007199254740993},"children":[],"state":null,"unknown":false}`}
	c := jobClient(f)
	got, _, e := c.Jobs.Get(context.Background(), jobTestID)
	if e != nil || string(got["data"]) != `{"progress":0,"large":9007199254740993}` || string(got["state"]) != "null" || string(got["children"]) != "[]" || string(got["unknown"]) != "false" {
		t.Fatal(got, e)
	}
	for _, id := range []string{"", "..", "a/b", "a?x", "a#x", "a%2fb", "a\\b", "a\n", "é"} {
		g, r, err := c.Jobs.Get(context.Background(), id)
		if err == nil || g != nil || r != nil || f.calls != 1 {
			t.Fatal(id, g, r, err, f.calls)
		}
	}
}
func TestJobsGetFailures(t *testing.T) {
	for _, body := range []string{"null", "[]", "1", `{"uuid":`, ""} {
		f := &jobWire{t: t, code: 200, body: body}
		g, _, e := jobClient(f).Jobs.Get(context.Background(), jobTestID)
		if e == nil || g != nil || f.calls != 1 || f.closed != 1 {
			t.Fatal(body, g, e)
		}
	}
	for _, code := range []int{403, 404, 500} {
		f := &jobWire{t: t, code: code, body: `[{"error_message":"denied","error_point":"job","error_type":"synthetic"}]`}
		g, r, e := jobClient(f).Jobs.Get(context.Background(), jobTestID)
		var api *ErrorResponse
		if g != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || len(api.Errors) != 1 || api.Errors[0].Message != "denied" || f.calls != 1 || f.closed != 1 {
			t.Fatal(g, r, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	f := &jobWire{t: t, err: sentinel}
	g, _, e := jobClient(f).Jobs.Get(context.Background(), jobTestID)
	if g != nil || !errors.Is(e, sentinel) || f.calls != 1 {
		t.Fatal(g, e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f = &jobWire{t: t}
	g, _, e = jobClient(f).Jobs.Get(ctx, jobTestID)
	if g != nil || !errors.Is(e, context.Canceled) || f.calls != 1 {
		t.Fatal(g, e)
	}
}
