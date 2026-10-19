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

const driveUsersID = "9876be5b-db07-4bc3-99cb-00c7ff55ec19"

type driveUsersGetWire struct {
	t           *testing.T
	body        []byte
	code, calls int
	err         error
}

func (f *driveUsersGetWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/driveusers/"+driveUsersID+"/" {
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
	return &http.Response{StatusCode: f.code, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(f.body)), Request: q}, nil
}
func driveUsersGetClient(f *driveUsersGetWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func TestDriveUsersGetRawPublic(t *testing.T) {
	b, e := os.ReadFile("testdata/drive_users_get_response.json")
	if e != nil {
		t.Fatal(e)
	}
	f := &driveUsersGetWire{t: t, body: b, code: 200}
	c := driveUsersGetClient(f)
	got, r, e := c.DriveUsers.GetRaw(context.Background(), driveUsersID)
	if e != nil || r == nil || r.StatusCode != 200 || r.Meta != nil || f.calls != 1 {
		t.Fatalf("result %v", e)
	}
	encoded, _ := json.Marshal(got)
	if !reflect.DeepEqual(aclJSON(t, encoded), aclJSON(t, b)) {
		t.Fatal("full public response")
	}
	again, _, e := c.DriveUsers.GetRaw(context.Background(), driveUsersID)
	if e != nil || f.calls != 2 {
		t.Fatal(e)
	}
	got["amount"][0] = '!'
	if got["amount"][0] != '!' || again["amount"][0] == '!' {
		t.Fatal("ownership")
	}

}
func TestDriveUsersGetRawPresence(t *testing.T) {
	for _, s := range []string{`{}`, `{"amount":null,"meta":{},"tags":[],"future":9007199254740993,"decimal":1.2300}`, `{"amount":[],"rules":null}`} {
		f := &driveUsersGetWire{t: t, body: []byte(s), code: 200}
		got, _, e := driveUsersGetClient(f).DriveUsers.GetRaw(context.Background(), driveUsersID)
		if e != nil {
			t.Fatal(e)
		}
		b, _ := json.Marshal(got)
		if !reflect.DeepEqual(aclJSON(t, b), aclJSON(t, []byte(s))) {
			t.Fatal("presence/precision")
		}
	}
}
func TestDriveUsersGetRawRefusals(t *testing.T) {
	for _, id := range []string{"", "..", "a/b", "a?x", "a#x", "%2f", "a\n"} {
		f := &driveUsersGetWire{t: t}
		got, r, e := driveUsersGetClient(f).DriveUsers.GetRaw(context.Background(), id)
		if e == nil || got != nil || r != nil || f.calls != 0 {
			t.Fatal("identifier dispatch")
		}
	}
	for _, s := range []string{"null", "[]", "false", "1", "\"text\"", "{"} {
		f := &driveUsersGetWire{t: t, body: []byte(s), code: 200}
		got, _, e := driveUsersGetClient(f).DriveUsers.GetRaw(context.Background(), driveUsersID)
		if e == nil || got != nil || f.calls != 1 {
			t.Fatal("malformed response")
		}
	}
}
func TestDriveUsersGetRawErrors(t *testing.T) {
	sentinel := errors.New("synthetic transport")
	f := &driveUsersGetWire{t: t, err: sentinel}
	got, _, e := driveUsersGetClient(f).DriveUsers.GetRaw(context.Background(), driveUsersID)
	if !errors.Is(e, sentinel) || got != nil || f.calls != 1 {
		t.Fatal("transport")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f = &driveUsersGetWire{t: t}
	got, _, e = driveUsersGetClient(f).DriveUsers.GetRaw(ctx, driveUsersID)
	if !errors.Is(e, context.Canceled) || got != nil {
		t.Fatal("cancel")
	}
	f = &driveUsersGetWire{t: t, code: 403, body: []byte(`[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`)}
	got, r, e := driveUsersGetClient(f).DriveUsers.GetRaw(context.Background(), driveUsersID)
	if e == nil || got != nil || r == nil || r.StatusCode != 403 || f.calls != 1 {
		t.Fatal("http error")
	}
}

func TestDriveUsersGetRawTypedError(t *testing.T) {
	f := &driveUsersGetWire{t: t, code: 403, body: []byte(`[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`)}
	_, r, e := driveUsersGetClient(f).DriveUsers.GetRaw(context.Background(), driveUsersID)
	var api *ErrorResponse
	if !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0].Type != "forbidden" || api.Errors[0].Message != "synthetic refusal" || api.Errors[0].Point != "" {
		t.Fatal("typed fields")
	}
}
