package cloudsigma

import (
	"context"
	"net/http"
)

// NotificationContactCreateRequest contains the publicly documented writable
// fields for contact creation. Nil pointers omit fields; non-nil pointers send
// their value, including an explicitly empty string.
// Format, required-field and account policies are validated by the server.
type NotificationContactCreateRequest struct {
	Email *string `json:"email,omitempty"`
	Name  *string `json:"name,omitempty"`
	Phone *string `json:"phone,omitempty"`
}

// CreateRaw posts a contact collection and retains the complete raw response
// envelope. Nil input is sent as an empty objects array; no local batch-size or
// required-field rules are invented. HTTP 201 acknowledges creation, not message
// delivery or account confirmation. No retries or follow-up requests are made.
// Redirects are refused with ErrNotificationContactsCreateRedirect. The initial
// POST outcome is unknown; callers must not blindly retry.
// Docs: https://docs.cloudsigma.com/en/latest/notification_contacts.html#creating
func (s *NotificationContactsService) CreateRaw(ctx context.Context, contacts []NotificationContactCreateRequest) (*NotificationContacts, *Response, error) {
	if contacts == nil {
		contacts = []NotificationContactCreateRequest{}
	}
	body := struct {
		Objects []NotificationContactCreateRequest `json:"objects"`
	}{Objects: contacts}
	req, err := s.client.NewRequest(http.MethodPost, "notification_contacts/", body)
	if err != nil {
		return nil, nil, err
	}
	value := new(NotificationContacts)
	resp, err := s.client.doResourceAction(ctx, req, value, ErrNotificationContactsCreateRedirect)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
