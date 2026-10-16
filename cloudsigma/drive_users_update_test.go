package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

type driveUsersUpdateBody struct {
	io.Reader
	closed int
}

func (b *driveUsersUpdateBody) Close() error { b.closed++; return nil }

type driveUsersUpdateWire struct {
	t                   *testing.T
	amount, code, calls int
	body                *driveUsersUpdateBody
	err                 error
}

func (f *driveUsersUpdateWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "PUT" || q.URL.RequestURI() != "/api/2.0/driveusers/e26fd531-7336-40e7-9a6f-db42e7a6b9f2/" {
		f.t.Fatal("method/route")
	}
	var body map[string]json.RawMessage
	if e := json.NewDecoder(q.Body).Decode(&body); e != nil {
		f.t.Fatal(e)
	}
	var amount int
	if e := json.Unmarshal(body["amount"], &amount); e != nil || amount != f.amount || len(body) != 1 {
		f.t.Fatalf("body=%s", body)
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: f.body, Request: q}, nil
}
func driveUsersUpdateClient(f *driveUsersUpdateWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}

const driveUsersUpdateID = "e26fd531-7336-40e7-9a6f-db42e7a6b9f2"

func TestDriveUsersUpdatePublic(t *testing.T) {
	// Five is the documented request. Zero and negative values only characterize
	// lossless forwarding, not valid provider ranges or server acceptance.
	for _, amount := range []int{5, 0, -1} {
		b := &driveUsersUpdateBody{Reader: bytes.NewBufferString(`{"objects":[{"amount":5,"drive":{"resource_uri":"/api/2.0/drives/e26fd531-7336-40e7-9a6f-db42e7a6b9f2/","uuid":"e26fd531-7336-40e7-9a6f-db42e7a6b9f2"}}],"extension":9007199254740993}`)}
		f := &driveUsersUpdateWire{t: t, amount: amount, code: 200, body: b}
		out, r, e := driveUsersUpdateClient(f).DriveUsers.UpdateAmount(context.Background(), driveUsersUpdateID, amount)
		if e != nil || r == nil || r.StatusCode != 200 || f.calls != 1 || b.closed != 1 {
			t.Fatalf("response=%v error=%v calls=%d closed=%d", r, e, f.calls, b.closed)
		}
		if string(out["extension"]) != "9007199254740993" || string(out["objects"]) != `[{"amount":5,"drive":{"resource_uri":"/api/2.0/drives/e26fd531-7336-40e7-9a6f-db42e7a6b9f2/","uuid":"e26fd531-7336-40e7-9a6f-db42e7a6b9f2"}}]` {
			t.Fatalf("response=%s", out)
		}
	}
}
func TestDriveUsersUpdateRefusals(t *testing.T) {
	for _, id := range []string{"", "..", "a/b", "a?x", "a#x", "%2f", "a\n", "a\\b"} {
		f := &driveUsersUpdateWire{t: t}
		out, r, e := driveUsersUpdateClient(f).DriveUsers.UpdateAmount(context.Background(), id, 5)
		if e == nil || out != nil || r != nil || f.calls != 0 {
			t.Fatalf("invalid %q dispatched", id)
		}
	}
}
func TestDriveUsersUpdateHTTPError(t *testing.T) {
	// Synthetic errors characterize the existing decoder, not provider refusal codes.
	for _, code := range []int{403, 404, 409, 500} {
		b := &driveUsersUpdateBody{Reader: bytes.NewBufferString(`[{"error_type":"synthetic","error_message":"refused","error_point":null}]`)}
		f := &driveUsersUpdateWire{t: t, amount: 5, code: code, body: b}
		out, r, e := driveUsersUpdateClient(f).DriveUsers.UpdateAmount(context.Background(), driveUsersUpdateID, 5)
		var er *ErrorResponse
		if !errors.As(e, &er) || out != nil || r == nil || r.StatusCode != code || f.calls != 1 || b.closed != 1 {
			t.Fatalf("response=%v error=%v", r, e)
		}
	}
}
func TestDriveUsersUpdateMalformed(t *testing.T) {
	for _, body := range []string{"null", "[]", "{", ""} {
		b := &driveUsersUpdateBody{Reader: bytes.NewBufferString(body)}
		f := &driveUsersUpdateWire{t: t, amount: 5, code: 200, body: b}
		out, _, e := driveUsersUpdateClient(f).DriveUsers.UpdateAmount(context.Background(), driveUsersUpdateID, 5)
		if e == nil || out != nil || f.calls != 1 || b.closed != 1 {
			t.Fatalf("body=%q error=%v", body, e)
		}
	}
}
func TestDriveUsersUpdateTransport(t *testing.T) {
	sentinel := errors.New("synthetic transport")
	f := &driveUsersUpdateWire{t: t, amount: 5, err: sentinel}
	out, r, e := driveUsersUpdateClient(f).DriveUsers.UpdateAmount(context.Background(), driveUsersUpdateID, 5)
	if !errors.Is(e, sentinel) || out != nil || r != nil || f.calls != 1 {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f = &driveUsersUpdateWire{t: t, amount: 5}
	out, r, e = driveUsersUpdateClient(f).DriveUsers.UpdateAmount(ctx, driveUsersUpdateID, 5)
	if !errors.Is(e, context.Canceled) || out != nil || r != nil || f.calls > 1 {
		t.Fatal(e)
	}
}
