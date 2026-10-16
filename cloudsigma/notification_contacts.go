package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// NotificationContactsService provides the public notification contact listing.
// Docs: https://docs.cloudsigma.com/en/latest/notification_contacts.html
type NotificationContactsService service

// NotificationContacts is a raw JSON object preserving decimal strings, numeric lexemes,
// unknown fields and absent-versus-null values without float conversion.
// The complete collection envelope is retained. Raw preservation does not validate
// contact semantics or imply server acceptance. Each call returns independent data.
type NotificationContacts map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (b *NotificationContacts) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("notification_contacts response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*b = fields
	return nil
}

// ListRaw reads one GET /notification_contacts/ collection envelope. It preserves meta,
// objects and unknown envelope fields without projecting them into Response.Meta.
// No pagination traversal, contact validation or
// schema validation is performed. Existing Client.Do decoding/error rules apply.
func (s *NotificationContactsService) ListRaw(ctx context.Context, opts *ListOptions) (*NotificationContacts, *Response, error) {
	query := url.Values{}
	if opts != nil {
		query.Set("limit", strconv.Itoa(opts.Limit))
		if opts.Offset != 0 {
			query.Set("offset", strconv.Itoa(opts.Offset))
		}
	}
	path := "notification_contacts/"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(NotificationContacts)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
