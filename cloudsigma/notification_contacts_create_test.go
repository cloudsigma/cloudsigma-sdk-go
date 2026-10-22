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

type notificationContactCreateWire struct {
	t                   *testing.T
	body                string
	code, calls, closed int
	want                string
	err                 error
}
type notificationContactCreateBody struct {
	*strings.Reader
	w *notificationContactCreateWire
}

func (b *notificationContactCreateBody) Close() error { b.w.closed++; return nil }
func (w *notificationContactCreateWire) RoundTrip(q *http.Request) (*http.Response, error) {
	w.calls++
	if q.Method != "POST" || q.URL.RequestURI() != "/custom/api/2.0/notification_contacts/" {
		w.t.Fatalf("route %s %s", q.Method, q.URL)
	}
	if q.Header.Get("Authorization") != "Bearer synthetic" {
		w.t.Fatal("authentication")
	}
	b, e := io.ReadAll(q.Body)
	if e != nil {
		w.t.Fatal(e)
	}
	var got, want interface{}
	if json.Unmarshal(b, &got) != nil || json.Unmarshal([]byte(w.want), &want) != nil || !reflect.DeepEqual(got, want) {
		w.t.Fatalf("body %s wanted %s", b, w.want)
	}
	if q.Header.Get("Content-Type") != "application/json" {
		w.t.Fatal("content type")
	}
	if e := q.Context().Err(); e != nil {
		return nil, e
	}
	if w.err != nil {
		return nil, w.err
	}
	return &http.Response{StatusCode: w.code, Header: make(http.Header), Body: &notificationContactCreateBody{strings.NewReader(w.body), w}, Request: q}, nil
}
func notificationContactCreateClient(w *notificationContactCreateWire) *sdk.Client {
	return sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: w}))
}
func TestNotificationContactsCreateEnvelope(t *testing.T) {
	for _, input := range [][]sdk.NotificationContactCreateRequest{nil, {}, {{Email: contactString("a@example.invalid"), Name: contactString("Name"), Phone: contactString("+1 123")}, {}}} {
		want := `{"objects":[]}`
		if len(input) > 0 {
			want = `{"objects":[{"email":"a@example.invalid","name":"Name","phone":"+1 123"},{}]}`
		}
		body := `{"objects":[{"email":"a@example.invalid","main":null,"name":"Name","phone":"+1123","resource_uri":"/api/2.0/notification_contacts/id/","uuid":"id","future":123456789012345678901}],"future":null}`
		w := &notificationContactCreateWire{t: t, code: 201, body: body, want: want}
		v, r, e := notificationContactCreateClient(w).NotificationContacts.CreateRaw(context.Background(), input)
		if e != nil || v == nil || r == nil || r.StatusCode != 201 || r.Meta != nil || w.calls != 1 || w.closed != 1 {
			t.Fatalf("result %v %v %v", v, r, e)
		}
		if string((*v)["objects"]) != `[{"email":"a@example.invalid","main":null,"name":"Name","phone":"+1123","resource_uri":"/api/2.0/notification_contacts/id/","uuid":"id","future":123456789012345678901}]` || string((*v)["future"]) != "null" {
			t.Fatal("raw preservation", v)
		}
	}
}
func TestNotificationContactsCreateErrors(t *testing.T) {
	for _, code := range []int{401, 403, 404, 409, 429, 500} {
		w := &notificationContactCreateWire{t: t, code: code, want: `{"objects":[]}`, body: `[{"error_type":"forbidden","error_message":"synthetic refusal","error_point":null}]`}
		v, r, e := notificationContactCreateClient(w).NotificationContacts.CreateRaw(context.Background(), nil)
		var api *sdk.ErrorResponse
		if v != nil || r == nil || r.StatusCode != code || !errors.As(e, &api) || api.Response != r || len(api.Errors) != 1 || api.Errors[0].Message != "synthetic refusal" || w.calls != 1 || w.closed != 1 {
			t.Fatalf("HTTP%d: %v", code, e)
		}
	}
	for _, body := range []string{`[]`, `null`, `{"objects":`} {
		w := &notificationContactCreateWire{t: t, code: 201, want: `{"objects":[]}`, body: body}
		v, r, e := notificationContactCreateClient(w).NotificationContacts.CreateRaw(context.Background(), nil)
		if v != nil || r != nil || e == nil || w.closed != 1 {
			t.Fatalf("invalid response %s %v", body, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	w := &notificationContactCreateWire{t: t, err: sentinel, want: `{"objects":[]}`}
	v, r, e := notificationContactCreateClient(w).NotificationContacts.CreateRaw(context.Background(), nil)
	if v != nil || r != nil || !errors.Is(e, sentinel) || w.calls != 1 {
		t.Fatal("transport", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w = &notificationContactCreateWire{t: t, want: `{"objects":[]}`}
	v, r, e = notificationContactCreateClient(w).NotificationContacts.CreateRaw(ctx, nil)
	if v != nil || r != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("context", e)
	}
}

func contactString(s string) *string { return &s }

// All three documented writable fields independently support absent, empty and populated.
func TestNotificationContactsCreatePresence(t *testing.T) {
	for a := 0; a < 3; a++ {
		for b := 0; b < 3; b++ {
			for c := 0; c < 3; c++ {
				t.Run(fmt.Sprintf("email%d/name%d/phone%d", a, b, c), func(t *testing.T) {
					states := []int{a, b, c}
					names := []string{"email", "name", "phone"}
					values := []string{"a@example.invalid", "Name", "+1 123"}
					var fields [3]*string
					wantObject := map[string]string{}
					for i, state := range states {
						if state != 0 {
							value := ""
							if state == 2 {
								value = values[i]
							}
							fields[i] = contactString(value)
							wantObject[names[i]] = value
						}
					}
					input := []sdk.NotificationContactCreateRequest{{Email: fields[0], Name: fields[1], Phone: fields[2]}}
					before := input[0]
					snapshots := [3]string{}
					for i, p := range fields {
						if p != nil {
							snapshots[i] = *p
						}
					}
					want, _ := json.Marshal(struct {
						Objects []map[string]string `json:"objects"`
					}{[]map[string]string{wantObject}})
					w := &notificationContactCreateWire{t: t, code: 201, body: `{"objects":[]}`, want: string(want)}
					_, _, err := notificationContactCreateClient(w).NotificationContacts.CreateRaw(context.Background(), input)
					if err != nil || w.calls != 1 || w.closed != 1 {
						t.Fatal(err, w.calls, w.closed)
					}
					if input[0] != before || input[0].Email != fields[0] || input[0].Name != fields[1] || input[0].Phone != fields[2] {
						t.Fatal("caller pointers changed")
					}
					for i, p := range fields {
						if p != nil && *p != snapshots[i] {
							t.Fatal("caller pointee changed")
						}
					}
				})
			}
		}
	}
}
