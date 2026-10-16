package cloudsigma

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
)

type driveUsersDeleteBody struct {
	io.Reader
	closed int
}

func (b *driveUsersDeleteBody) Close() error { b.closed++; return nil }

type driveUsersDeleteWire struct {
	t           *testing.T
	code, calls int
	body        *driveUsersDeleteBody
	err         error
}

func (f *driveUsersDeleteWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != http.MethodDelete || q.URL.RequestURI() != "/api/2.0/driveusers/e26fd531-7336-40e7-9a6f-db42e7a6b9f2/" {
		f.t.Fatal("method/route")
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
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: f.body, Request: q}, nil
}
func driveUsersDeleteClient(f *driveUsersDeleteWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}

const driveUsersDeleteID = "e26fd531-7336-40e7-9a6f-db42e7a6b9f2"

func TestDriveUsersDeletePublic(t *testing.T) {
	b := &driveUsersDeleteBody{Reader: bytes.NewReader(nil)}
	f := &driveUsersDeleteWire{t: t, code: 204, body: b}
	r, e := driveUsersDeleteClient(f).DriveUsers.Delete(context.Background(), driveUsersDeleteID)
	if e != nil || r == nil || r.StatusCode != 204 || f.calls != 1 || b.closed != 1 {
		t.Fatalf("response=%v error=%v calls=%d closed=%d", r, e, f.calls, b.closed)
	}
}
func TestDriveUsersDeleteRefusals(t *testing.T) {
	for _, id := range []string{"", "..", "a/b", "a?x", "a#x", "%2f", "a\n", "a\\b"} {
		f := &driveUsersDeleteWire{t: t}
		r, e := driveUsersDeleteClient(f).DriveUsers.Delete(context.Background(), id)
		if e == nil || r != nil || f.calls != 0 {
			t.Fatalf("invalid identifier %q dispatched", id)
		}
	}
}
func TestDriveUsersDeleteHTTPError(t *testing.T) {
	// Synthetic errors characterize the shared SDK decoder, not a documented
	// mounted-drive status or resolution of the common error-envelope question.
	for _, code := range []int{403, 404, 409, 500} {
		b := &driveUsersDeleteBody{Reader: bytes.NewBufferString(`[{"error_type":"synthetic","error_message":"refused","error_point":null}]`)}
		f := &driveUsersDeleteWire{t: t, code: code, body: b}
		r, e := driveUsersDeleteClient(f).DriveUsers.Delete(context.Background(), driveUsersDeleteID)
		var er *ErrorResponse
		if !errors.As(e, &er) || r == nil || r.StatusCode != code || f.calls != 1 || b.closed != 1 {
			t.Fatalf("code=%d response=%v error=%v", code, r, e)
		}
	}
}
func TestDriveUsersDeleteTransport(t *testing.T) {
	sentinel := errors.New("synthetic")
	f := &driveUsersDeleteWire{t: t, err: sentinel}
	r, e := driveUsersDeleteClient(f).DriveUsers.Delete(context.Background(), driveUsersDeleteID)
	if !errors.Is(e, sentinel) || r != nil || f.calls != 1 {
		t.Fatal("transport", r, e, f.calls)
	}
}
func TestDriveUsersDeleteCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &driveUsersDeleteWire{t: t}
	r, e := driveUsersDeleteClient(f).DriveUsers.Delete(ctx, driveUsersDeleteID)
	if !errors.Is(e, context.Canceled) || r != nil || f.calls > 1 {
		t.Fatal("cancellation", r, e, f.calls)
	}
}
