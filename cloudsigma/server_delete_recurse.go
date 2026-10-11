package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// ServerDeleteRecurse selects the documented attached-drive deletion scope.
type ServerDeleteRecurse string

const (
	ServerDeleteAllDrives ServerDeleteRecurse = "all_drives"
	ServerDeleteDisks     ServerDeleteRecurse = "disks"
	ServerDeleteCDROMs    ServerDeleteRecurse = "cdroms"
)

// DeleteRecursive explicitly requests deletion of a server and selected attached
// drives. The existing Delete method requests server-only deletion. HTTP 204
// acknowledges deletion started, not completion. No polling or followup delete
// is performed; ownership, partial-failure and additional cascade guarantees are
// unspecified. Empty/unknown modes and unsafe route identifiers are refused as
// SDK policy, not a public UUID grammar. Legacy Delete is unchanged.
// Docs: https://docs.cloudsigma.com/en/latest/servers.html#deleting
func (s *ServersService) DeleteRecursive(ctx context.Context, id string, mode ServerDeleteRecurse) (*Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, err
	}
	switch mode {
	case ServerDeleteAllDrives, ServerDeleteDisks, ServerDeleteCDROMs:
	default:
		return nil, fmt.Errorf("invalid server deletion recurse mode")
	}
	req, err := s.client.NewRequest(http.MethodDelete, fmt.Sprintf("servers/%s/?recurse=%s", id, mode), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}
