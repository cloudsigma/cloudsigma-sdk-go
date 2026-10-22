package cloudsigma_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type notificationContactUpdateWire struct {
	t                   *testing.T
	body, want          string
	code, calls, closed int
	err                 error
}
type notificationContactUpdateBody struct {
	*strings.Reader
	w *notificationContactUpdateWire
}

func (b *notificationContactUpdateBody) Close() error { b.w.closed++; return nil }
func (w *notificationContactUpdateWire) RoundTrip(q *http.Request) (*http.Response, error) {
	w.calls++
	if q.Method != "PUT" || q.URL.RequestURI() != "/custom/api/2.0/notification_contacts/synthetic-contact/" {
		w.t.Fatalf("route %s %s", q.Method, q.URL)
	}
	if q.Header.Get("Authorization") != "Bearer synthetic" || q.Header.Get("Content-Type") != "application/json" || q.Header.Get("Accept") != "application/json" {
		w.t.Fatal("headers")
	}
	b, e := io.ReadAll(q.Body)
	if e != nil {
		w.t.Fatal(e)
	}
	var got, want map[string]string
	if json.Unmarshal(b, &got) != nil || json.Unmarshal([]byte(w.want), &want) != nil || !reflect.DeepEqual(got, want) {
		w.t.Fatalf("body %s wanted %s", b, w.want)
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if w.err != nil {
		return nil, w.err
	}
	return &http.Response{StatusCode: w.code, Header: make(http.Header), Body: &notificationContactUpdateBody{strings.NewReader(w.body), w}, Request: q}, nil
}
func notificationContactUpdateClient(w *notificationContactUpdateWire) *sdk.Client {
	return sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
}
func TestNotificationContactsUpdatePresence(t *testing.T) {
	for a := 0; a < 3; a++ {
		for b := 0; b < 3; b++ {
			t.Run(fmt.Sprintf("email%d/name%d", a, b), func(t *testing.T) {
				var fields [2]*string
				want := map[string]string{}
				for i, state := range []int{a, b} {
					if state != 0 {
						value := ""
						if state == 2 {
							value = []string{"a@example.invalid", "New Name"}[i]
						}
						fields[i] = &value
						want[[]string{"email", "name"}[i]] = value
					}
				}
				input := sdk.NotificationContactUpdateRequest{Email: fields[0], Name: fields[1]}
				before := input
				var snapshots [2]string
				for i, p := range fields {
					if p != nil {
						snapshots[i] = *p
					}
				}
				encoded, _ := json.Marshal(want)
				w := &notificationContactUpdateWire{t: t, code: 200, body: `{"email":"a@example.invalid","main":null,"name":"New Name","phone":"+1123","pk":"id","resource_uri":"/api/2.0/notification_contacts/id/","uuid":"id","future":{"n":123456789012345678901}}`, want: string(encoded)}
				c := notificationContactUpdateClient(w)
				v, r, e := c.NotificationContacts.UpdateRaw(context.Background(), "synthetic-contact", input)
				if e != nil || v == nil || r == nil || r.StatusCode != 200 || r.Meta != nil || w.calls != 1 || w.closed != 1 {
					t.Fatalf("result %v %v %v", v, r, e)
				}
				if string((*v)["future"]) != `{"n":123456789012345678901}` || string((*v)["main"]) != "null" || string((*v)["phone"]) != `"+1123"` || len(*v) != 8 {
					t.Fatal("raw preservation", v)
				}
				if input != before {
					t.Fatal("input pointers changed")
				}
				for i, p := range fields {
					if p != nil && *p != snapshots[i] {
						t.Fatal("input value changed")
					}
				}
				(*v)["future"][0] = '['
				if (*v)["future"][0] != '[' {
					t.Fatal("mutation did not occur")
				}
				v2, _, e := c.NotificationContacts.UpdateRaw(context.Background(), "synthetic-contact", input)
				if e != nil || string((*v2)["future"]) != `{"n":123456789012345678901}` || w.calls != 2 || w.closed != 2 {
					t.Fatal("independent result", e)
				}
			})
		}
	}
}
func TestNotificationContactsUpdateRefusals(t *testing.T) {
	for _, id := range []string{"", ".", "..", "a/b", "a?x", "a#x", "%2f", "a\\b", "a\n", "a\x00"} {
		w := &notificationContactUpdateWire{t: t}
		v, r, e := notificationContactUpdateClient(w).NotificationContacts.UpdateRaw(context.Background(), id, sdk.NotificationContactUpdateRequest{})
		if e == nil || v != nil || r != nil || w.calls != 0 {
			t.Fatalf("identifier %q", id)
		}
	}
	for _, endpoint := range []string{"https://example.invalid/custom/api/2.0/", "not-an-endpoint"} {
		w := &notificationContactUpdateWire{t: t}
		c := sdk.NewClient(nil, sdk.WithEndpoint(endpoint), sdk.WithHTTPClient(&http.Client{Transport: w}))
		v, r, e := c.NotificationContacts.UpdateRaw(context.Background(), "synthetic-contact", sdk.NotificationContactUpdateRequest{})
		if e == nil || v != nil || r != nil || w.calls != 0 {
			t.Fatal("request setup")
		}
	}
}
func TestNotificationContactsUpdateErrors(t *testing.T) {
	for _, code := range []int{400, 401, 403, 404, 409, 429, 500} {
		w := &notificationContactUpdateWire{t: t, code: code, want: `{}`, body: `[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`}
		v, r, e := notificationContactUpdateClient(w).NotificationContacts.UpdateRaw(context.Background(), "synthetic-contact", sdk.NotificationContactUpdateRequest{})
		var api *sdk.ErrorResponse
		if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0].Message != "synthetic refusal" || w.calls != 1 || w.closed != 1 {
			t.Fatalf("HTTP%d %v", code, e)
		}
	}
	for _, body := range []string{`[]`, `null`, `{"name":`, ``} {
		w := &notificationContactUpdateWire{t: t, code: 200, want: `{}`, body: body}
		v, r, e := notificationContactUpdateClient(w).NotificationContacts.UpdateRaw(context.Background(), "synthetic-contact", sdk.NotificationContactUpdateRequest{})
		if v != nil || r != nil || e == nil || w.calls != 1 || w.closed != 1 {
			t.Fatalf("malformed %q %v", body, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	w := &notificationContactUpdateWire{t: t, err: sentinel, want: `{}`}
	v, r, e := notificationContactUpdateClient(w).NotificationContacts.UpdateRaw(context.Background(), "synthetic-contact", sdk.NotificationContactUpdateRequest{})
	if v != nil || r != nil || !errors.Is(e, sentinel) || w.calls != 1 || w.closed != 0 {
		t.Fatal("transport", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w = &notificationContactUpdateWire{t: t, want: `{}`}
	v, r, e = notificationContactUpdateClient(w).NotificationContacts.UpdateRaw(ctx, "synthetic-contact", sdk.NotificationContactUpdateRequest{})
	if v != nil || r != nil || !errors.Is(e, context.Canceled) || w.calls > 1 || w.closed != 0 {
		t.Fatal("context", e)
	}
}
