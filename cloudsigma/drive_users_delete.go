package cloudsigma

import (
	"context"
	"net/http"
)

// Delete deletes the RDS user-count record identified by the drive UUID, not
// the drive itself. The public API refuses deletion while the drive is mounted;
// this method leaves that check to the server and preserves HTTP errors.
// It sends one request, with no unlinking, retries, polling or 404 suppression.
// A successful HTTP response is not independent confirmation of remote absence.
// The identifier follows the SDK conservative segment policy, not UUID grammar.
// Docs: https://docs.cloudsigma.com/en/latest/drives.html
func (s *DriveUsersService) Delete(ctx context.Context, id string) (*Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, err
	}
	req, err := s.client.NewRequest(http.MethodDelete, "driveusers/"+id+"/", nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}
