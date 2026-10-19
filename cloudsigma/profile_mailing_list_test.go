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

type mailingWire struct {
	t                   *testing.T
	enabled             bool
	calls, closed, code int
	body                string
	err                 error
}

func (f *mailingWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "PUT" || q.URL.Path != "/api/2.0/profile/" || q.URL.RawQuery != "" || q.URL.Fragment != "" {
		f.t.Fatal("route", q.URL)
	}
	if q.Header.Get("Content-Type") != "application/json" {
		f.t.Fatal("content type")
	}
	raw, e := io.ReadAll(q.Body)
	if e != nil {
		f.t.Fatal(e)
	}
	var got map[string]interface{}
	if e = json.Unmarshal(raw, &got); e != nil {
		f.t.Fatal(e)
	}
	if !reflect.DeepEqual(got, map[string]interface{}{"mailing_list": f.enabled}) {
		f.t.Fatalf("presence/readonly echo: %s", raw)
	}
	if e = q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: &mailingBody{strings.NewReader(f.body), f}, Request: q}, nil
}

type mailingBody struct {
	*strings.Reader
	owner *mailingWire
}

func (b *mailingBody) Close() error { b.owner.closed++; return nil }
func mailingClient(f *mailingWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func TestProfileMailingListPresence(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		body := `{"uuid":"synthetic-profile","company":"unchanged","mailing_list":false}`
		if enabled {
			body = `{"uuid":"synthetic-profile","company":"unchanged","mailing_list":true}`
		}
		f := &mailingWire{t: t, enabled: enabled, code: 200, body: body}
		p, r, e := mailingClient(f).Profile.UpdateMailingList(context.Background(), enabled)
		if e != nil || r == nil || r.StatusCode != 200 || p == nil || p.MailingList != enabled || p.UUID != "synthetic-profile" || p.Company != "unchanged" || f.calls != 1 || f.closed != 1 {
			t.Fatal(p, r, e, f.calls, f.closed)
		}
	}
	// Legacy compatibility is intentional: its false omission is not a normative
	// representation of opting out. The additive method supplies that missing facet.
	raw, e := json.Marshal(&ProfileUpdateRequest{Profile: &Profile{MailingList: false}})
	if e != nil || strings.Contains(string(raw), "mailing_list") {
		t.Fatal(string(raw), e)
	}
}
func TestProfileMailingListFailures(t *testing.T) {
	for _, code := range []int{403, 404, 500} {
		f := &mailingWire{t: t, code: code, body: `[{"error_message":"denied","error_point":"mailing_list","error_type":"synthetic"}]`}
		p, r, e := mailingClient(f).Profile.UpdateMailingList(context.Background(), false)
		var api *ErrorResponse
		if p != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || len(api.Errors) != 1 || api.Errors[0].Message != "denied" || f.calls != 1 || f.closed != 1 {
			t.Fatal(p, r, e)
		}
	}
	f := &mailingWire{t: t, code: 200, body: `{"mailing_list":`}
	p, _, e := mailingClient(f).Profile.UpdateMailingList(context.Background(), false)
	if p != nil || e == nil || f.calls != 1 || f.closed != 1 {
		t.Fatal(p, e)
	}
	sentinel := errors.New("synthetic transport")
	f = &mailingWire{t: t, err: sentinel}
	p, _, e = mailingClient(f).Profile.UpdateMailingList(context.Background(), false)
	if p != nil || !errors.Is(e, sentinel) || f.calls != 1 {
		t.Fatal(p, e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f = &mailingWire{t: t}
	p, _, e = mailingClient(f).Profile.UpdateMailingList(ctx, false)
	if p != nil || !errors.Is(e, context.Canceled) || f.calls != 1 {
		t.Fatal(p, e)
	}
}
