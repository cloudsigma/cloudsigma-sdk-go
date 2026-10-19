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

type driveUsersCreateBody struct {
	io.Reader
	closed int
}

func (b *driveUsersCreateBody) Close() error { b.closed++; return nil }

type driveUsersCreateWire struct {
	t                   *testing.T
	amount, code, calls int
	body                *driveUsersCreateBody
	err                 error
}

func (f *driveUsersCreateWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/driveusers/" {
		f.t.Fatal("method/route")
	}
	var body map[string]json.RawMessage
	if e := json.NewDecoder(q.Body).Decode(&body); e != nil {
		f.t.Fatal(e)
	}
	var amount int
	if e := json.Unmarshal(body["amount"], &amount); e != nil || amount != f.amount || len(body) != 2 || string(body["drive"]) != `"e26fd531-7336-40e7-9a6f-db42e7a6b9f2"` {
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
func driveUsersCreateClient(f *driveUsersCreateWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}

const driveUsersCreateID = "e26fd531-7336-40e7-9a6f-db42e7a6b9f2"

func TestDriveUsersCreatePublic(t *testing.T) {
	// Five is the documented request. Zero and negative values only characterize
	// lossless forwarding, not valid provider ranges or server acceptance.
	for _, amount := range []int{5, 0, -1} {
		b := &driveUsersCreateBody{Reader: bytes.NewBufferString(`{"objects":[{"amount":5,"drive":{"resource_uri":"/api/2.0/drives/e26fd531-7336-40e7-9a6f-db42e7a6b9f2/","uuid":"e26fd531-7336-40e7-9a6f-db42e7a6b9f2"}}],"extension":9007199254740993}`)}
		f := &driveUsersCreateWire{t: t, amount: amount, code: 201, body: b}
		out, r, e := driveUsersCreateClient(f).DriveUsers.Create(context.Background(), driveUsersCreateID, amount)
		if e != nil || r == nil || r.StatusCode != 201 || f.calls != 1 || b.closed != 1 {
			t.Fatalf("response=%v error=%v calls=%d closed=%d", r, e, f.calls, b.closed)
		}
		if string(out["extension"]) != "9007199254740993" || string(out["objects"]) != `[{"amount":5,"drive":{"resource_uri":"/api/2.0/drives/e26fd531-7336-40e7-9a6f-db42e7a6b9f2/","uuid":"e26fd531-7336-40e7-9a6f-db42e7a6b9f2"}}]` {
			t.Fatalf("response=%s", out)
		}
	}
}
func TestDriveUsersCreateRefusals(t *testing.T) {
	for _, id := range []string{"", "..", "a/b", "a?x", "a#x", "%2f", "a\n", "a\\b"} {
		f := &driveUsersCreateWire{t: t}
		out, r, e := driveUsersCreateClient(f).DriveUsers.Create(context.Background(), id, 5)
		if e == nil || out != nil || r != nil || f.calls != 0 {
			t.Fatalf("invalid %q dispatched", id)
		}
	}
}
func TestDriveUsersCreateHTTPError(t *testing.T) {
	// Synthetic errors characterize the existing decoder, not provider refusal codes.
	for _, code := range []int{403, 404, 409, 500} {
		b := &driveUsersCreateBody{Reader: bytes.NewBufferString(`[{"error_type":"synthetic","error_message":"refused","error_point":null}]`)}
		f := &driveUsersCreateWire{t: t, amount: 5, code: code, body: b}
		out, r, e := driveUsersCreateClient(f).DriveUsers.Create(context.Background(), driveUsersCreateID, 5)
		var er *ErrorResponse
		if !errors.As(e, &er) || out != nil || r == nil || r.StatusCode != code || f.calls != 1 || b.closed != 1 {
			t.Fatalf("response=%v error=%v", r, e)
		}
	}
}
func TestDriveUsersCreateMalformed(t *testing.T) {
	for _, body := range []string{"null", "[]", "{", ""} {
		b := &driveUsersCreateBody{Reader: bytes.NewBufferString(body)}
		f := &driveUsersCreateWire{t: t, amount: 5, code: 201, body: b}
		out, _, e := driveUsersCreateClient(f).DriveUsers.Create(context.Background(), driveUsersCreateID, 5)
		if e == nil || out != nil || f.calls != 1 || b.closed != 1 {
			t.Fatalf("body=%q error=%v", body, e)
		}
	}
}
func TestDriveUsersCreateTransport(t *testing.T) {
	sentinel := errors.New("synthetic transport")
	f := &driveUsersCreateWire{t: t, amount: 5, err: sentinel}
	out, r, e := driveUsersCreateClient(f).DriveUsers.Create(context.Background(), driveUsersCreateID, 5)
	if !errors.Is(e, sentinel) || out != nil || r != nil || f.calls != 1 {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f = &driveUsersCreateWire{t: t, amount: 5}
	out, r, e = driveUsersCreateClient(f).DriveUsers.Create(ctx, driveUsersCreateID, 5)
	if !errors.Is(e, context.Canceled) || out != nil || r != nil || f.calls > 1 {
		t.Fatal(e)
	}
}
