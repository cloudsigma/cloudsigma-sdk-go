package cloudsigma

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
)

type resizePublicWire struct {
	t        *testing.T
	calls    int
	response string
}

func (f *resizePublicWire) RoundTrip(q *http.Request) (*http.Response, error) {
	f.calls++
	if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/drives/df170528-c34d-4009-a806-c7d85c78cad8/action/?do=resize" {
		f.t.Fatalf("wrong wire %s %s", q.Method, q.URL.RequestURI())
	}
	raw, e := io.ReadAll(q.Body)
	if e != nil {
		f.t.Fatal(e)
	}
	// Legacy projection characterization, NOT normative public contract.
	// Future lossless opt-in work needs separate expectations; revise this
	// expectation if legacy representation intentionally changes.
	expected := `{"media":"disk","name":"test_drive_1","owner":{"resource_uri":"/api/2.0/user/5b4a69a3-8e78-4c45-a8ba-8b13f0895e23/","uuid":"5b4a69a3-8e78-4c45-a8ba-8b13f0895e23"},"resource_uri":"/api/2.0/drives/df170528-c34d-4009-a806-c7d85c78cad8/","runtime":{"storage_type":"dssd"},"size":4294967296,"status":"creating","storage_type":"dssd","uuid":"df170528-c34d-4009-a806-c7d85c78cad8"}`
	var got, want map[string]interface{}
	if e = json.Unmarshal(raw, &got); e != nil {
		f.t.Fatal(e)
	}
	if e = json.Unmarshal([]byte(expected), &want); e != nil {
		f.t.Fatal(e)
	}
	if !reflect.DeepEqual(got, want) {
		f.t.Fatalf("unexpected wire payload %s", raw)
	}
	return &http.Response{StatusCode: http.StatusAccepted, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.response)), Request: q}, nil
}

func TestDrivesResizeLegacyProjectionMock202(t *testing.T) {
	// Exact full-definition public example retained as input. The current SDK
	// omits zero/empty and unmodeled fields: this test does NOT prove server
	// acceptance of that lossy projection or complete full-definition support.
	request, e := os.ReadFile("resize_public_request.json")
	if e != nil {
		t.Fatal(e)
	}
	response, e := os.ReadFile("resize_public_response.json")
	if e != nil {
		t.Fatal(e)
	}
	var drive Drive
	if e = json.Unmarshal(request, &drive); e != nil {
		t.Fatal(e)
	}
	f := &resizePublicWire{t: t, response: string(response)}
	c := NewClient(NewTokenCredentialsProvider("synthetic-only"), WithHTTPClient(&http.Client{Transport: f}))
	result, r, e := c.Drives.Resize(context.Background(), drive.UUID, &DriveUpdateRequest{Drive: &drive})
	if e != nil || r == nil || r.StatusCode != http.StatusAccepted || len(result) != 1 || f.calls != 1 {
		t.Fatalf("result=%v response=%v err=%v calls=%d", result, r, e, f.calls)
	}
	if result[0].UUID != drive.UUID || result[0].Status != "resizing" || result[0].Size != 2147483648 || result[0].Name != "test_drive_1" || result[0].StorageType != "dssd" {
		t.Fatalf("documented response lost: %+v", result[0])
	}
	//202 and documented pre-resize size do not establish completion.
}
