package cloudsigma

import (
	"context"
	"encoding/json"
)

// ServerDriveOptions is a request-only drive attachment using the scalar UUID
// shown in the public attach-drive example. It does not imply that legacy
// nested drive references are rejected by the API. Nil BootOrder omits the
// optional setting; a nonnil pointer preserves zero. No default is injected.
// Device and DevChannel are required by the public schema.
// https://docs.cloudsigma.com/en/latest/servers.html#attach-a-drive
// https://docs.cloudsigma.com/en/latest/servers.html#schema
type ServerDriveOptions struct {
	BootOrder  *int64 `json:"boot_order,omitempty"`
	DevChannel string `json:"dev_channel"`
	Device     string `json:"device"`
	Drive      string `json:"drive"`
}

// UpdateDriveOptions sends only the supplied ordered drive attachment list.
// This is a server-definition edit, not an append operation. Callers must supply
// the intended list; this method does not read or merge existing attachments.
// Nil lists are rejected; empty lists encode [] (representation, not a promise
// of server acceptance). Changes on running servers are ignored by the API.
// The result is a response snapshot, not proof of runtime attachment completion.
// Callers must not mutate options or pointed-to values concurrently with this call.
func (s *ServersService) UpdateDriveOptions(ctx context.Context, id string, options []ServerDriveOptions) (*Server, *Response, error) {
	if options == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}
	body, err := json.Marshal(options)
	if err != nil {
		return nil, nil, err
	}
	return s.UpdateDefinition(ctx, id, ServerDefinition{"drives": body})
}
