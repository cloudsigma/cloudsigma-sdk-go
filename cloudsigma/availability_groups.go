package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
)

var availabilityUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type availabilityIDs []string

func (a *availabilityIDs) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if raw == nil {
		return fmt.Errorf("availability group must be an array")
	}
	out := make([]string, len(raw))
	for i, v := range raw {
		if err := json.Unmarshal(v, &out[i]); err != nil {
			return err
		}
		if !availabilityUUID.MatchString(out[i]) {
			return fmt.Errorf("availability group contains invalid UUID")
		}
	}
	*a = out
	return nil
}

type availabilityGroups [][]string

func (a *availabilityGroups) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if raw == nil {
		return fmt.Errorf("availability groups must be an array")
	}
	out := make([][]string, len(raw))
	for i, v := range raw {
		var ids availabilityIDs
		if err := json.Unmarshal(v, &ids); err != nil {
			return err
		}
		out[i] = ids
	}
	*a = out
	return nil
}
func (c *Client) availabilityRead(ctx context.Context, kind, id string, single bool) (availabilityGroups, availabilityIDs, *Response, error) {
	path := kind + "/availability_groups/"
	if single {
		if id == "" {
			return nil, nil, nil, ErrEmptyArgument
		}
		if !availabilityUUID.MatchString(id) {
			return nil, nil, nil, fmt.Errorf("invalid availability UUID")
		}
		path += id + "/"
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	if single {
		var ids availabilityIDs
		resp, err := c.Do(ctx, req, &ids)
		if err != nil {
			return nil, nil, resp, err
		}
		return nil, ids, resp, nil
	}
	var groups availabilityGroups
	resp, err := c.Do(ctx, req, &groups)
	if err != nil {
		return nil, nil, resp, err
	}
	return groups, nil, resp, nil
}

// ListAvailabilityGroups returns running server UUIDs grouped by compute host.
// Docs: https://docs.cloudsigma.com/en/latest/availability_groups.html
func (s *ServersService) ListAvailabilityGroups(ctx context.Context) ([][]string, *Response, error) {
	g, _, r, e := s.client.availabilityRead(ctx, "servers", "", false)
	return g, r, e
}

// GetAvailabilityGroup returns the server UUIDs sharing the given server's host.
// A non-running server may return an empty array. UUID input is validated locally.
func (s *ServersService) GetAvailabilityGroup(ctx context.Context, id string) ([]string, *Response, error) {
	_, g, r, e := s.client.availabilityRead(ctx, "servers", id, true)
	return g, r, e
}

// ListAvailabilityGroups returns drive UUIDs grouped by storage host.
// Docs: https://docs.cloudsigma.com/en/latest/availability_groups.html
func (s *DrivesService) ListAvailabilityGroups(ctx context.Context) ([][]string, *Response, error) {
	g, _, r, e := s.client.availabilityRead(ctx, "drives", "", false)
	return g, r, e
}

// GetAvailabilityGroup returns drive UUIDs sharing the given drive's storage host.
// UUID input is validated locally; no placement guarantee or mutation is implied.
func (s *DrivesService) GetAvailabilityGroup(ctx context.Context, id string) ([]string, *Response, error) {
	_, g, r, e := s.client.availabilityRead(ctx, "drives", id, true)
	return g, r, e
}
