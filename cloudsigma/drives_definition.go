package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// DriveDefinition is an opt-in JSON object representation for a full drive
// definition. Raw values preserve unknown fields, numbers and explicit null,
// false, zero and empty collections. It does not establish field writability.
// Unmarshaling non-object JSON fails; JSON null produces a nil definition,
// which ResizeDefinition rejects. Duplicate keys follow encoding/json rules.
type DriveDefinition map[string]json.RawMessage

// ResizeDefinition submits a full definition without the legacy Drive model's
// omitempty projection. Callers must provide the complete definition required
// by the public API; this method validates JSON shape, not server field policy.
// A 202 response means acceptance, not completed resizing. The returned Drive
// values retain the existing response decoder and its modeled-field limits.
// Identifiers use a conservative SDK URL-safety policy: ASCII letters, digits,
// hyphen and underscore only. This is not a claim about all API-valid IDs.
// Callers must not concurrently modify definition or its raw byte slices.
//
// https://cloudsigma-docs.readthedocs.io/en/latest/drives.html#resizing-update-or-fail
// Redirects are refused with ErrDriveResizeDefinitionRedirect; the initial POST
// outcome is unknown. Callers must not blindly retry.
func (s *DrivesService) ResizeDefinition(ctx context.Context, uuid string, definition DriveDefinition) ([]Drive, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}
	// Reject URL syntax rather than escaping it: no percent decoding, path
	// traversal, query/fragment injection or server-dependent interpretation.
	for _, c := range uuid {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_') {
			return nil, nil, fmt.Errorf("invalid drive identifier: unsafe URL segment")
		}
	}
	if definition == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}
	// Copy all values before dispatch; neither this method nor NewRequest mutates
	// the caller map. Invalid RawMessages fail locally, before any HTTP request.
	snapshot := make(DriveDefinition, len(definition))
	for key, value := range definition {
		if !json.Valid(value) {
			return nil, nil, fmt.Errorf("invalid drive definition JSON value")
		}
		snapshot[key] = append(json.RawMessage(nil), value...)
	}
	path := fmt.Sprintf("%v/%v/action/?do=resize", drivesBasePath, uuid)
	req, err := s.client.NewRequest(http.MethodPost, path, snapshot)
	if err != nil {
		return nil, nil, err
	}
	root := new(drivesRoot)
	resp, err := s.client.doResourceAction(ctx, req, root, ErrDriveResizeDefinitionRedirect)
	if err != nil {
		return nil, resp, err
	}
	return root.values(), resp, nil
}
