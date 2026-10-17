package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// DriveUsersService accesses RDS drive user-count records. It does not manage
// accounts or permissions, or provision drives or servers.
type DriveUsersService service

// GetRaw retrieves the RDS user-count record identified by the drive UUID.
// Raw values retain field presence and numeric tokens and are caller-owned.
// Null/nonobject responses are refused; field types are not schema-validated.
// The identifier uses the SDK conservative segment policy, not public UUID grammar.
// No links are followed and no resource or account mutation is performed.
// Docs: https://docs.cloudsigma.com/en/latest/drives.html#rds-drive-detail-single-drive
func (s *DriveUsersService) GetRaw(ctx context.Context, id string) (map[string]json.RawMessage, *Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodGet, "driveusers/"+id+"/", nil)
	if err != nil {
		return nil, nil, err
	}
	var out map[string]json.RawMessage
	resp, err := s.client.Do(ctx, req, &out)
	if err != nil {
		return nil, resp, err
	}
	if out == nil {
		return nil, resp, fmt.Errorf("expected RDS drive user-count response object")
	}
	return out, resp, nil
}
