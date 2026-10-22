package cloudsigma_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type notificationContactDeleteWire struct {
	t                   *testing.T
	body                string
	code, calls, closed int
	err                 error
}
type notificationContactDeleteBody struct {
	*strings.Reader
	w *notificationContactDeleteWire
}

func (b *notificationContactDeleteBody) Close() error { b.w.closed++; return nil }
func (w *notificationContactDeleteWire) RoundTrip(q *http.Request) (*http.Response, error) {
	w.calls++
	if q.Method != "DELETE" || q.URL.RequestURI() != "/custom/api/2.0/notification_contacts/synthetic-contact/" {
		w.t.Fatalf("route %s %s", q.Method, q.URL)
	}
	if q.Header.Get("Authorization") != "Bearer synthetic" {
		w.t.Fatal("authentication")
	}
	if q.Body != nil {
		b, e := io.ReadAll(q.Body)
		if e != nil || len(b) != 0 {
			w.t.Fatal("request body")
		}
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if w.err != nil {
		return nil, w.err
	}
	return &http.Response{StatusCode: w.code, Header: make(http.Header), Body: &notificationContactDeleteBody{strings.NewReader(w.body), w}, Request: q}, nil
}
func notificationContactDeleteClient(w *notificationContactDeleteWire) *sdk.Client {
	return sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
}
func TestNotificationContactsDeleteAcknowledgment(t *testing.T) {
	w := &notificationContactDeleteWire{t: t, code: 204}
	r, e := notificationContactDeleteClient(w).NotificationContacts.Delete(context.Background(), "synthetic-contact")
	if e != nil || r == nil || r.StatusCode != 204 || r.Meta != nil || w.calls != 1 || w.closed != 1 {
		t.Fatalf("acknowledgment %v %v %+v", r, e, w)
	}
	// One request, no retry or polling; no notification delivery is inferred.
}
func TestNotificationContactsDeleteRefusals(t *testing.T) {
	for _, id := range []string{"", ".", "..", "a/b", "a?x", "a#x", "%2f", "a\n"} {
		w := &notificationContactDeleteWire{t: t}
		r, e := notificationContactDeleteClient(w).NotificationContacts.Delete(context.Background(), id)
		if e == nil || r != nil || w.calls != 0 {
			t.Fatalf("invalid identifier %q", id)
		}
	}
}
func TestNotificationContactsDeleteErrors(t *testing.T) {
	for _, code := range []int{401, 403, 404, 409, 429, 500} {
		w := &notificationContactDeleteWire{t: t, code: code, body: `[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`}
		r, e := notificationContactDeleteClient(w).NotificationContacts.Delete(context.Background(), "synthetic-contact")
		var api *sdk.ErrorResponse
		if r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0].Message != "synthetic refusal" || w.calls != 1 || w.closed != 1 {
			t.Fatalf("HTTP%d: %v", code, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	w := &notificationContactDeleteWire{t: t, err: sentinel}
	r, e := notificationContactDeleteClient(w).NotificationContacts.Delete(context.Background(), "synthetic-contact")
	if r != nil || !errors.Is(e, sentinel) || w.calls != 1 {
		t.Fatal("transport", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w = &notificationContactDeleteWire{t: t}
	r, e = notificationContactDeleteClient(w).NotificationContacts.Delete(ctx, "synthetic-contact")
	if r != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("context", e)
	}
}
