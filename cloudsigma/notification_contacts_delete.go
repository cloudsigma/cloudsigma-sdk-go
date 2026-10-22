package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// Delete requests deletion of a notification contact. The public API documents
// HTTP 204 as no errors. The main contact cannot be deleted; that restriction
// is enforced by the server, not inferred from the identifier locally.
// No polling or retries are performed, and notification-delivery completion is
// not implied. Unsafe route identifiers are refused as SDK policy, not a public
// UUID grammar.
// Docs: https://docs.cloudsigma.com/en/latest/notification_contacts.html#deleting
func (s *NotificationContactsService) Delete(ctx context.Context, id string) (*Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, err
	}
	req, err := s.client.NewRequest(http.MethodDelete, fmt.Sprintf("notification_contacts/%s/", id), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}
