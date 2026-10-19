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

func TestNotificationContactsListRaw(t *testing.T) {
	fixtures := []string{
		// Populated public notification-contact listing, retaining every example field.
		`{"meta":{"limit":0,"offset":0,"total_count":1},"objects":[{"email":"some_email_1@example.com","main":true,"name":"Some Name","phone":"","resource_uri":"/api/2.0/notification_contacts/e13c4b8c-68c8-4163-bc00-2a43a86e5937/","uuid":"e13c4b8c-68c8-4163-bc00-2a43a86e5937"}]}`,
		`{"meta":{"future":9007199254740993},"objects":[{"main":null,"future":[3,2,1]}],"extension":null}`,
		`{}`, `{"objects":null}`, `{"objects":[]}`, `{"meta":null}`,
	}
	for _, tc := range []struct {
		opts *ListOptions
		uri  string
	}{
		{nil, "/api/2.0/notification_contacts/"},
		{&ListOptions{}, "/api/2.0/notification_contacts/?limit=0"},
		{&ListOptions{Limit: 2, Offset: 3}, "/api/2.0/notification_contacts/?limit=2&offset=3"},
	} {
		for _, body := range fixtures {
			before, _ := json.Marshal(tc.opts)
			calls := 0
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
				calls++
				if q.Method != "GET" || q.URL.RequestURI() != tc.uri {
					t.Fatalf("wire: %s %s", q.Method, q.URL.RequestURI())
				}
				if q.Body != nil && q.Body != http.NoBody {
					b, e := io.ReadAll(q.Body)
					if e != nil || len(b) != 0 {
						t.Fatal("body", e)
					}
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
			})}))
			a, r, e := c.NotificationContacts.ListRaw(context.Background(), tc.opts)
			if e != nil || a == nil || r == nil || r.StatusCode != 200 || calls != 1 {
				t.Fatal("list", e, calls)
			}
			var expected map[string]json.RawMessage
			if err := json.Unmarshal([]byte(body), &expected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(map[string]json.RawMessage(*a), expected) {
				t.Fatal("envelope/precision/null/order")
			}
			after, _ := json.Marshal(tc.opts)
			if string(before) != string(after) {
				t.Fatal("options mutated")
			}
			(*a)["injected"] = json.RawMessage(`true`)
			if raw := (*a)["objects"]; len(raw) > 0 {
				raw[0] = 'X'
				if (*a)["objects"][0] != 'X' {
					t.Fatal("mutation failed")
				}
			}
			b, _, e := c.NotificationContacts.ListRaw(context.Background(), tc.opts)
			if e != nil || calls != 2 || !reflect.DeepEqual(map[string]json.RawMessage(*b), expected) {
				t.Fatal("ownership", e)
			}
		}
	}
}

func TestNotificationContactsErrors(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `1`, `"x"`, `{`, ``} {
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}))
		a, r, e := c.NotificationContacts.ListRaw(context.Background(), nil)
		if a != nil || r != nil || e == nil {
			t.Fatal("malformed convention", body, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	for _, code := range []int{403, 404, 500, 0} {
		calls := 0
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			calls++
			if code == 0 {
				return nil, sentinel
			}
			return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[{"error_type":"permission","error_message":"synthetic denied","error_point":null}]`)), Request: q}, nil
		})}))
		a, r, e := c.NotificationContacts.ListRaw(context.Background(), nil)
		if a != nil || e == nil || calls != 1 {
			t.Fatal("error", e)
		}
		if code == 0 {
			if !errors.Is(e, sentinel) || r != nil {
				t.Fatal("transport", e)
			}
		} else {
			var api *ErrorResponse
			if !errors.As(e, &api) || r == nil || r.StatusCode != code || len(api.Errors) != 1 || api.Errors[0].Message != "synthetic denied" || api.Errors[0].Type != "permission" || api.Errors[0].Point != "" || api.Response != r {
				t.Fatal("typed error", e)
			}
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) { return nil, q.Context().Err() })}))
	if _, _, e := c.NotificationContacts.ListRaw(ctx, nil); !errors.Is(e, context.Canceled) {
		t.Fatal("cancel", e)
	}
}
