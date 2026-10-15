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

func driveRawClient(f func(*http.Request) (*http.Response, error)) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(f)}))
}
func driveRawReply(q *http.Request, code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}
}
func TestDriveRawPreservation(t *testing.T) {
	// Nonempty nested members are synthetic preservation probes, not schema claims.
	for _, body := range []string{
		`{"affinities":[{"future":{"n":9007199254740993}}],"grantees":[{"uuid":"synthetic","unknown":null}],"permissions":[{"scope":["one","two"]}],"jobs":[{"uuid":"job-1","resource_uri":"/api/2.0/jobs/job-1/"}],"runtime":{"is_snapshotable":false,"is_downloadable":true,"is_ephemeral":false,"snapshots_allocated_size":0,"storage_type":"dssd","unknown":1.23000000000000000001},"size":9007199254740993,"licenses":[{"unknown":{"null":null}}],"meta":{"nested":{"ordered":[2,1]}}}`,
		`{}`, `{"owner":null,"runtime":null,"size":null,"meta":null}`, `{"owner":{},"runtime":{},"size":0,"meta":{},"jobs":[],"allow_multimount":false}`, `{"unknown":{"n":1e99},"name":""}`,
	} {
		calls := 0
		c := driveRawClient(func(q *http.Request) (*http.Response, error) {
			calls++
			if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/drives/drive-1/" {
				t.Fatal(q)
			}
			if q.Body != nil {
				b, e := io.ReadAll(q.Body)
				if e != nil || len(b) != 0 {
					t.Fatal("body", e)
				}
			}
			return driveRawReply(q, 200, body), nil
		})
		got, r, e := c.Drives.GetRaw(context.Background(), "drive-1")
		if e != nil || r == nil || r.StatusCode != 200 || calls != 1 {
			t.Fatal(got, r, e, calls)
		}
		var want map[string]json.RawMessage
		if e = json.Unmarshal([]byte(body), &want); e != nil {
			t.Fatal(e)
		}
		if !reflect.DeepEqual(map[string]json.RawMessage(*got), want) {
			t.Fatalf("full nested preservation: %s", body)
		}
		(*got)["injected"] = json.RawMessage(`true`)
		mutated := 0
		for k, v := range *got {
			if k != "injected" && len(v) > 0 {
				v[0] = 'X'
				if (*got)[k][0] != 'X' {
					t.Fatal("mutation failed")
				}
				mutated++
			}
		}
		if string((*got)["injected"]) != "true" || len(want) > 0 && mutated == 0 {
			t.Fatal("mutation not exercised")
		}
		fresh, _, e := c.Drives.GetRaw(context.Background(), "drive-1")
		if e != nil || calls != 2 || !reflect.DeepEqual(map[string]json.RawMessage(*fresh), want) {
			t.Fatal("fresh ownership", e)
		}
	}
}
func TestDriveRawErrors(t *testing.T) {
	for _, code := range []int{401, 403, 404, 500} {
		calls := 0
		c := driveRawClient(func(q *http.Request) (*http.Response, error) {
			calls++
			return driveRawReply(q, code, `[{"error_message":"synthetic","error_point":"drives","error_type":"denied"}]`), nil
		})
		v, r, e := c.Drives.GetRaw(context.Background(), "drive-1")
		var typed *ErrorResponse
		if v != nil || calls != 1 || !errors.As(e, &typed) || typed.Response != r || r.StatusCode != code || len(typed.Errors) != 1 || typed.Errors[0].Message != "synthetic" || typed.Errors[0].Point != "drives" || typed.Errors[0].Type != "denied" {
			t.Fatal(v, r, e)
		}
	}
	for _, body := range []string{"", `null`, `[]`, `42`, `"x"`, `{`} {
		calls := 0
		c := driveRawClient(func(q *http.Request) (*http.Response, error) { calls++; return driveRawReply(q, 200, body), nil })
		v, r, e := c.Drives.GetRaw(context.Background(), "drive-1")
		if v != nil || r != nil || e == nil || calls != 1 {
			t.Fatal("malformed", body, v, r, e, calls)
		}
	}
	sentinel := errors.New("synthetic transport")
	c := driveRawClient(func(q *http.Request) (*http.Response, error) { return nil, sentinel })
	if _, _, e := c.Drives.GetRaw(context.Background(), "drive-1"); !errors.Is(e, sentinel) {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c = driveRawClient(func(q *http.Request) (*http.Response, error) { return nil, q.Context().Err() })
	if _, _, e := c.Drives.GetRaw(ctx, "drive-1"); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	calls := 0
	c = driveRawClient(func(q *http.Request) (*http.Response, error) { calls++; return nil, sentinel })
	for _, id := range []string{"", "..", "a/b", "a?x", "a#x", "a%2fb", "a\n", "é"} {
		v, r, e := c.Drives.GetRaw(context.Background(), id)
		if v != nil || r != nil || !errors.Is(e, ErrEmptyArgument) || calls != 0 {
			t.Fatal(id, v, r, e, calls)
		}
	}
}
