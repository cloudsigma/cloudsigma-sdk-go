package cloudsigma

import (
	"context"
	"net/http"
)

// NotificationContactUpdateRequest contains fields shown in the public contact
// editing request. Nil omits a field; a non-nil pointer sends its value, including
// an empty string. Required-field and format rules are validated by the server.
// Response-only fields and metadata are not forwarded by this request type.
type NotificationContactUpdateRequest struct {
	Email *string `json:"email,omitempty"`
	Name  *string `json:"name,omitempty"`
}

// UpdateRaw edits one notification contact and returns the complete raw singleton
// object, not a collection envelope. The public API discourages changing the main
// user's contact email because account-email changes also change that contact.
// Absent/empty field effects remain server-owned; no merge semantics are promised.
// Unsafe identifiers are refused as SDK policy, not a public UUID grammar.
// HTTP 200 acknowledges the edit, not notification delivery. No retries, polling
// or follow-up requests are made. Existing Client.Do decoding/error rules apply.
// Docs: https://docs.cloudsigma.com/en/latest/notification_contacts.html#editing
func (s *NotificationContactsService) UpdateRaw(ctx context.Context, id string, input NotificationContactUpdateRequest) (*NotificationContacts, *Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodPut, "notification_contacts/"+id+"/", input)
	if err != nil {
		return nil, nil, err
	}
	value := new(NotificationContacts)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
