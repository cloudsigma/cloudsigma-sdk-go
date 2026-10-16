package cloudsigma

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type libraryCloneAckTransport struct {
	t        *testing.T
	body     string
	status   int
	payload  bool
	calls    int
	response *http.Response
	err      error
}

func (f *libraryCloneAckTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.calls++
	if req.Method != "POST" || req.URL.String() != "https://library-clone.invalid/libdrives/source/action/?do=clone" {
		f.t.Fatalf("unexpected request: %s %s", req.Method, req.URL)
	}
	var wire []byte
	if req.Body != nil {
		var err error
		wire, err = io.ReadAll(req.Body)
		if err != nil {
			f.t.Fatal(err)
		}
	}
	if f.payload {
		var got map[string]json.RawMessage
		if err := json.Unmarshal(wire, &got); err != nil {
			f.t.Fatal(err)
		}
		want := map[string]json.RawMessage{"name": json.RawMessage(`"copy"`), "size": json.RawMessage(`4096`), "uuid": json.RawMessage(`""`)}
		if !reflect.DeepEqual(got, want) {
			f.t.Fatalf("request payload = %s", wire)
		}
	} else if len(wire) != 0 {
		f.t.Fatalf("optional request body = %s", wire)
	}
	if f.err != nil {
		return nil, f.err
	}
	f.response = &http.Response{StatusCode: f.status, Header: http.Header{"X-Request-Id": []string{"clone-fixture"}}, Body: io.NopCloser(strings.NewReader(f.body)), Request: req}
	return f.response, nil
}

// Synthetic acknowledgments qualify SDK handling, not cloud clone completion.
func TestLibraryCloneAcknowledgment(t *testing.T) {
	for _, body := range []string{`{"objects":[]}`, `{}`, `{"objects":null}`, `null`} {
		for _, status := range []int{201, 202} {
			for _, payload := range []bool{false, true} {
				name := body + "/" + http.StatusText(status)
				if payload {
					name += "/payload"
				} else {
					name += "/no-payload"
				}
				t.Run(name, func(t *testing.T) {
					f := &libraryCloneAckTransport{t: t, body: body, status: status, payload: payload}
					c := libraryCloneAckClient(f)
					var request *LibraryDriveCloneRequest
					if payload {
						request = &LibraryDriveCloneRequest{LibraryDrive: &LibraryDrive{Name: "copy", Size: 4096}}
					}
					drive, resp, err := c.LibraryDrives.Clone(context.Background(), "source", request)
					if drive != nil || err == nil || err.Error() != "clone response contains no library drives" {
						t.Fatalf("empty acknowledgment: drive=%v error=%v", drive, err)
					}
					assertLibraryCloneResponse(t, f, resp)
				})
			}
		}
	}
}

func libraryCloneAckClient(f *libraryCloneAckTransport) *Client {
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: f}))
	c.baseURL, _ = url.Parse("https://library-clone.invalid/")
	return c
}

func assertLibraryCloneResponse(t *testing.T, f *libraryCloneAckTransport, resp *Response) {
	t.Helper()
	if f.calls != 1 || resp == nil || resp.Response != f.response || resp.StatusCode != f.status || resp.RequestID != "clone-fixture" {
		t.Fatalf("response identity/status/request ID lost: calls=%d response=%v", f.calls, resp)
	}
}

func TestLibraryCloneAcknowledgmentControls(t *testing.T) {
	for _, tc := range []struct{ name, body, uuid, free string }{
		{"normal", `{"objects":[{"uuid":"first","name":"copy"},{"uuid":"second"}]}`, "first", ""},
		{"numeric-license", `{"objects":[{"uuid":"first","licenses":[{"license":{"free":2}}]}]}`, "first", "2"},
		{"string-license", `{"objects":[{"uuid":"first","licenses":[{"license":{"free":"2"}}]}]}`, "first", "2"},
		// Preserve legacy identity policy; this repair only rejects an empty collection.
		{"missing-uuid", `{"objects":[{}]}`, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &libraryCloneAckTransport{t: t, body: tc.body, status: 202}
			drive, resp, err := libraryCloneAckClient(f).LibraryDrives.Clone(context.Background(), "source", nil)
			if err != nil || drive == nil || drive.UUID != tc.uuid {
				t.Fatalf("result: %v %v", drive, err)
			}
			assertLibraryCloneResponse(t, f, resp)
			if tc.free != "" && (len(drive.Licenses) != 1 || drive.Licenses[0].License.Free != tc.free) {
				t.Fatalf("license compatibility lost: %+v", drive.Licenses)
			}
			if tc.name == "normal" && drive.Name != "copy" {
				t.Fatalf("name lost: %+v", drive)
			}
		})
	}
	for _, body := range []string{`{`, `{"objects":false}`, ``} {
		t.Run("malformed/"+body, func(t *testing.T) {
			f := &libraryCloneAckTransport{t: t, body: body, status: 202}
			drive, resp, err := libraryCloneAckClient(f).LibraryDrives.Clone(context.Background(), "source", nil)
			if drive != nil || resp != nil || err == nil || f.calls != 1 {
				t.Fatalf("decode convention: %v %v %v", drive, resp, err)
			}
		})
	}
	t.Run("HTTP", func(t *testing.T) {
		f := &libraryCloneAckTransport{t: t, body: `[{"error_type":"forbidden","error_message":"denied","error_point":"drive"}]`, status: 403}
		drive, resp, err := libraryCloneAckClient(f).LibraryDrives.Clone(context.Background(), "source", nil)
		var apiErr *ErrorResponse
		if drive != nil || !errors.As(err, &apiErr) || apiErr.Response != resp || !reflect.DeepEqual(apiErr.Errors, []Error{{Type: "forbidden", Message: "denied", Point: "drive"}}) {
			t.Fatalf("HTTP error lost: %v %v", drive, err)
		}
		assertLibraryCloneResponse(t, f, resp)
	})
	t.Run("transport", func(t *testing.T) {
		sentinel := errors.New("synthetic transport failure")
		f := &libraryCloneAckTransport{t: t, err: sentinel}
		drive, resp, err := libraryCloneAckClient(f).LibraryDrives.Clone(context.Background(), "source", nil)
		if drive != nil || resp != nil || !errors.Is(err, sentinel) || f.calls != 1 {
			t.Fatalf("transport error lost: %v %v %v", drive, resp, err)
		}
	})
	t.Run("empty-source", func(t *testing.T) {
		f := &libraryCloneAckTransport{t: t}
		drive, resp, err := libraryCloneAckClient(f).LibraryDrives.Clone(context.Background(), "", nil)
		if drive != nil || resp != nil || err != ErrEmptyArgument || f.calls != 0 {
			t.Fatalf("argument convention: %v %v %v", drive, resp, err)
		}
	})
	t.Run("canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		f := &libraryCloneAckTransport{t: t, err: errors.New("transport cancellation")}
		drive, resp, err := libraryCloneAckClient(f).LibraryDrives.Clone(ctx, "source", nil)
		if drive != nil || resp != nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation lost: %v %v %v", drive, resp, err)
		}
	})
}
