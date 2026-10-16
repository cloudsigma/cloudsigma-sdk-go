package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ListRaw retrieves one complete RDS drive user-count collection envelope.
// Nil options omit pagination parameters; a nonnil zero Limit sends limit=0.
// Raw values retain presence and numeric tokens and are caller-owned. No links
// are followed and Response.Meta is not populated. Null/nonobject responses are
// refused; envelope and field schemas are not validated. This read does not
// create RDS resources or manage accounts or permissions.
// Docs: https://docs.cloudsigma.com/en/latest/drives.html#listing
func (s *DriveUsersService) ListRaw(ctx context.Context, opts *ListOptions) (map[string]json.RawMessage, *Response, error) {
	query := url.Values{}
	if opts != nil {
		query.Set("limit", strconv.Itoa(opts.Limit))
		if opts.Offset != 0 {
			query.Set("offset", strconv.Itoa(opts.Offset))
		}
	}
	path := "driveusers/"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	var out map[string]json.RawMessage
	resp, err := s.client.Do(ctx, req, &out)
	if err != nil {
		return nil, resp, err
	}
	if out == nil {
		return nil, resp, fmt.Errorf("expected RDS drive user-count collection response object")
	}
	return out, resp, nil
}
