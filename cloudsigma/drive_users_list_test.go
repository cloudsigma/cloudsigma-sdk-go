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

type driveUsersListWire struct {
	t           *testing.T
	body        []byte
	code, calls int
	err         error
	query       string
}

func (f *driveUsersListWire) path() string { return "/api/2.0/driveusers/" + f.query }
func (f *driveUsersListWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "GET" || q.URL.RequestURI() != f.path() {
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
func driveUsersListClient(f *driveUsersListWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
}
func TestDriveUsersListRawPublic(t *testing.T) {
	b, e := os.ReadFile("testdata/drive_users_list_response.json")
	if e != nil {
		t.Fatal(e)
	}
	f := &driveUsersListWire{t: t, body: b, code: 200}
	c := driveUsersListClient(f)
	got, r, e := c.DriveUsers.ListRaw(context.Background(), nil)
	if e != nil || r == nil || r.StatusCode != 200 || r.Meta != nil || f.calls != 1 {
		t.Fatalf("result %v", e)
	}
	encoded, _ := json.Marshal(got)
	if !reflect.DeepEqual(aclJSON(t, encoded), aclJSON(t, b)) {
		t.Fatal("full public response")
	}
	again, _, e := c.DriveUsers.ListRaw(context.Background(), nil)
	if e != nil || f.calls != 2 {
		t.Fatal(e)
	}
	got["objects"][0] = '!'
	if got["objects"][0] != '!' || again["objects"][0] == '!' {
		t.Fatal("ownership")
	}
	var objects []struct {
		Amount int `json:"amount"`
		Drive  struct {
			UUID string `json:"uuid"`
		} `json:"drive"`
	}
	if json.Unmarshal(again["objects"], &objects) != nil || len(objects) != 2 || objects[0].Amount != 2 || objects[1].Amount != 4 || objects[0].Drive.UUID != "9876be5b-db07-4bc3-99cb-00c7ff55ec19" || objects[1].Drive.UUID != "5a613e14-0e74-4f2c-b5b8-cf8e0403488a" {
		t.Fatal("populated public RDS records")
	}

}
func TestDriveUsersListRawPresence(t *testing.T) {
	for _, s := range []string{`{}`, `{"objects":null,"meta":{},"future":9007199254740993,"decimal":1.2300}`, `{"objects":[],"meta":null}`} {
		f := &driveUsersListWire{t: t, body: []byte(s), code: 200}
		got, _, e := driveUsersListClient(f).DriveUsers.ListRaw(context.Background(), nil)
		if e != nil {
			t.Fatal(e)
		}
		b, _ := json.Marshal(got)
		if !reflect.DeepEqual(aclJSON(t, b), aclJSON(t, []byte(s))) {
			t.Fatal("presence/precision")
		}
	}
}
func TestDriveUsersListRawRefusals(t *testing.T) {
	for _, s := range []string{"null", "[]", "false", "1", "\"text\"", "{"} {
		f := &driveUsersListWire{t: t, body: []byte(s), code: 200}
		got, _, e := driveUsersListClient(f).DriveUsers.ListRaw(context.Background(), nil)
		if e == nil || got != nil || f.calls != 1 {
			t.Fatal("malformed response")
		}
	}
}
func TestDriveUsersListRawErrors(t *testing.T) {
	sentinel := errors.New("synthetic transport")
	f := &driveUsersListWire{t: t, err: sentinel}
	got, _, e := driveUsersListClient(f).DriveUsers.ListRaw(context.Background(), nil)
	if !errors.Is(e, sentinel) || got != nil || f.calls != 1 {
		t.Fatal("transport")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f = &driveUsersListWire{t: t}
	got, _, e = driveUsersListClient(f).DriveUsers.ListRaw(ctx, nil)
	if !errors.Is(e, context.Canceled) || got != nil {
		t.Fatal("cancel")
	}
	f = &driveUsersListWire{t: t, code: 403, body: []byte(`[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`)}
	got, r, e := driveUsersListClient(f).DriveUsers.ListRaw(context.Background(), nil)
	if e == nil || got != nil || r == nil || r.StatusCode != 403 || f.calls != 1 {
		t.Fatal("http error")
	}
}

func TestDriveUsersListRawTypedError(t *testing.T) {
	f := &driveUsersListWire{t: t, code: 403, body: []byte(`[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`)}
	_, r, e := driveUsersListClient(f).DriveUsers.ListRaw(context.Background(), nil)
	var api *ErrorResponse
	if !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0].Type != "forbidden" || api.Errors[0].Message != "synthetic refusal" || api.Errors[0].Point != "" {
		t.Fatal("typed fields")
	}
}

func TestDriveUsersListRawPagination(t *testing.T) {
	for _, tc := range []struct {
		opts  *ListOptions
		query string
	}{{nil, ""}, {&ListOptions{}, "?limit=0"}, {&ListOptions{Limit: 2, Offset: 3}, "?limit=2&offset=3"}} {
		f := &driveUsersListWire{t: t, code: 200, body: []byte(`{"objects":[],"meta":{"next":"https://untrusted.invalid/"}}`), query: tc.query}
		_, r, e := driveUsersListClient(f).DriveUsers.ListRaw(context.Background(), tc.opts)
		if e != nil || r.Meta != nil || f.calls != 1 {
			t.Fatal("one page", e)
		}
	}
}
