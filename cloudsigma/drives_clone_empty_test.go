package cloudsigma

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type cloneFixtureTransport struct {
	body string
	err  error
}

func (f cloneFixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header),
		Body: io.NopCloser(strings.NewReader(f.body)), Request: req}, nil
}

func TestDrivesCloneEmptyEnvelope(t *testing.T) {
	for _, body := range []string{`{"objects":[]}`, `{"objects":null}`, `{}`} {
		t.Run(body, func(t *testing.T) {
			c := NewClient(NewTokenCredentialsProvider("fixture"), WithHTTPClient(&http.Client{
				Transport: cloneFixtureTransport{body: body},
			}))
			drive, resp, err := c.Drives.Clone(context.Background(), "source", nil)
			if drive != nil || resp == nil || err == nil || err.Error() != "clone response contains no drives" {
				t.Fatalf("unexpected result: drive=%v response=%v error=%v", drive, resp, err)
			}
		})
	}
}

func TestDrivesCloneFirstResultNumericLicense(t *testing.T) {
	body := `{"objects":[{"uuid":"first","licenses":[{"license":{"free":0}}]},{"uuid":"second"}]}`
	c := NewClient(NewTokenCredentialsProvider("fixture"), WithHTTPClient(&http.Client{
		Transport: cloneFixtureTransport{body: body},
	}))
	drive, resp, err := c.Drives.Clone(context.Background(), "source", nil)
	if err != nil || resp == nil || drive == nil || drive.UUID != "first" {
		t.Fatalf("unexpected clone result: %v %v %v", drive, resp, err)
	}
	if drive.Licenses[0].License.Free != "0" {
		t.Fatalf("numeric license response lost: %+v", drive.Licenses)
	}
}

func TestDrivesCloneTransportErrorPreserved(t *testing.T) {
	sentinel := errors.New("synthetic transport failure")
	c := NewClient(NewTokenCredentialsProvider("fixture"), WithHTTPClient(&http.Client{
		Transport: cloneFixtureTransport{err: sentinel},
	}))
	drive, _, err := c.Drives.Clone(context.Background(), "source", nil)
	if drive != nil || !errors.Is(err, sentinel) {
		t.Fatalf("transport failure not preserved: %v %v", drive, err)
	}
}
