package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Create defines an RDS drive user-count record using the documented drive UUID
// and amount members. Amount is included even when zero. The API owns count-range
// and Windows/RDS applicability validation. This does not provision a drive or
// server, mount it, retry, or poll. The drive identifier follows the existing SDK
// conservative segment policy, not public UUID grammar.
// The complete response object is returned without schema validation; raw field
// preservation and HTTP acknowledgment do not establish remote completion.
// Null/nonobject responses are refused; shared HTTP success handling is unchanged.
// Redirects are refused with ErrDriveUsersCreateRedirect. The initial POST
// outcome is unknown; callers must not blindly retry.
// Docs: https://docs.cloudsigma.com/en/latest/drives.html#creating-defining-rds-drive
func (s *DriveUsersService) Create(ctx context.Context, id string, amount int) (map[string]json.RawMessage, *Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, nil, err
	}
	body := struct {
		Drive  string `json:"drive"`
		Amount int    `json:"amount"`
	}{Drive: id, Amount: amount}
	req, err := s.client.NewRequest(http.MethodPost, "driveusers/", body)
	if err != nil {
		return nil, nil, err
	}
	var out map[string]json.RawMessage
	resp, err := s.client.doResourceAction(ctx, req, &out, ErrDriveUsersCreateRedirect)
	if err != nil {
		return nil, resp, err
	}
	if out == nil {
		return nil, resp, fmt.Errorf("expected RDS drive user-count create response object")
	}
	return out, resp, nil
}
