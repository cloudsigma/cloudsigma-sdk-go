package cloudsigma_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// Literal expectations follow the public contact creation example.
// Acknowledgment is not notification delivery or account confirmation.
func TestContactCreatePreparsePublicContract(t *testing.T) {
	for _, code := range []int{201, 200, 202} {
		calls := 0
		c := sdk.NewClient(sdk.NewTokenCredentialsProvider("contract-token"), sdk.WithEndpoint("https://example.invalid/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
			calls++
			if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/notification_contacts/" || q.Header.Get("Authorization") != "Bearer contract-token" {
				t.Error("public route/auth changed")
			}
			var body map[string][]map[string]string
			if e := json.NewDecoder(q.Body).Decode(&body); e != nil {
				t.Fatal(e)
			}
			_ = q.Body.Close()
			if !reflect.DeepEqual(body, map[string][]map[string]string{"objects": {{"email": "some_email_2@example.com", "name": "Some Name 2", "phone": "+1 123 555 1234"}}}) {
				t.Fatalf("documented payload changed: %#v", body)
			}
			return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[{"email":"some_email_2@example.com","main":null,"name":"Some Name 2","phone":"+11235551234","resource_uri":"/api/2.0/notification_contacts/0103f398-9f85-446a-a285-fe63cf9f0f28/","uuid":"0103f398-9f85-446a-a285-fe63cf9f0f28"}],"large":9007199254740993,"nullable":null}`)), Request: q}, nil
		})}))
		email, name, phone := "some_email_2@example.com", "Some Name 2", "+1 123 555 1234"
		v, r, e := c.NotificationContacts.CreateRaw(context.Background(), []sdk.NotificationContactCreateRequest{{Email: &email, Name: &name, Phone: &phone}})
		if e != nil || r == nil || r.StatusCode != code || v == nil || calls != 1 {
			t.Fatalf("acknowledgment changed: %v %v %v calls=%d", v, r, e, calls)
		}
		if string((*v)["objects"]) != `[{"email":"some_email_2@example.com","main":null,"name":"Some Name 2","phone":"+11235551234","resource_uri":"/api/2.0/notification_contacts/0103f398-9f85-446a-a285-fe63cf9f0f28/","uuid":"0103f398-9f85-446a-a285-fe63cf9f0f28"}]` || string((*v)["large"]) != "9007199254740993" || string((*v)["nullable"]) != "null" {
			t.Fatal("raw create acknowledgment changed")
		}
	}
}
