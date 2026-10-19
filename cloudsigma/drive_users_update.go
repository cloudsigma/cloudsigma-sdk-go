package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// UpdateAmount updates the user count of the RDS record identified by a drive
// UUID. It sends only the documented amount member, including an explicit zero.
// The API owns count-range and drive-applicability validation; this method does
// not create a drive, mount it, retry, or poll. The identifier uses the SDK's
// conservative segment policy, not public UUID grammar.
// The returned complete object envelope is caller-owned; raw preservation does
// not validate its schema or establish remote completion. Null/nonobject
// responses are refused. Existing shared HTTP success handling is unchanged.
// Docs: https://docs.cloudsigma.com/en/latest/drives.html#editing-updating-rds-drive
func (s *DriveUsersService) UpdateAmount(ctx context.Context, id string, amount int) (map[string]json.RawMessage, *Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, nil, err
	}
	body := struct {
		Amount int `json:"amount"`
	}{Amount: amount}
	req, err := s.client.NewRequest(http.MethodPut, "driveusers/"+id+"/", body)
	if err != nil {
		return nil, nil, err
	}
	var out map[string]json.RawMessage
	resp, err := s.client.Do(ctx, req, &out)
	if err != nil {
		return nil, resp, err
	}
	if out == nil {
		return nil, resp, fmt.Errorf("expected RDS drive user-count update response object")
	}
	return out, resp, nil
}
