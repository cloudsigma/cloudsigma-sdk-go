package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// Delete requests deletion of a single backfilling server. HTTP 204 acknowledges
// that deletion started, not that it completed. No polling, retries or recursive
// attached-drive deletion is performed. Unsafe route identifiers are refused as
// SDK policy, not a public UUID grammar.
// Docs: https://docs.cloudsigma.com/en/latest/backfilling.html#deleting
func (s *BServersService) Delete(ctx context.Context, id string) (*Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, err
	}
	req, err := s.client.NewRequest(http.MethodDelete, fmt.Sprintf("bservers/%s/", id), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}
