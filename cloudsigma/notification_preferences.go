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

// NotificationPreferencesService provides the public notification preference listing.
// Docs: https://docs.cloudsigma.com/en/latest/notification_preferences.html
type NotificationPreferencesService service

// NotificationPreferences is a raw JSON object preserving decimal strings, numeric lexemes,
// unknown fields and absent-versus-null values without float conversion.
// The complete collection envelope is retained. Raw preservation does not validate
// preference semantics or imply server acceptance. Each call returns independent data.
type NotificationPreferences map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (b *NotificationPreferences) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("notification_preferences response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*b = fields
	return nil
}

// ListRaw reads one GET /notification_preferences/ collection envelope. It preserves meta,
// objects and unknown envelope fields without projecting them into Response.Meta.
// No pagination traversal, preference validation or schema validation is performed.
// The documentation calls this a contact list in prose; this method follows the
// explicit preference route and preserves its response without reinterpretation.
// It does not implement preference updates or select their disputed success code. Existing Client.Do decoding/error rules apply.
func (s *NotificationPreferencesService) ListRaw(ctx context.Context, opts *ListOptions) (*NotificationPreferences, *Response, error) {
	query := url.Values{}
	if opts != nil {
		query.Set("limit", strconv.Itoa(opts.Limit))
		if opts.Offset != 0 {
			query.Set("offset", strconv.Itoa(opts.Offset))
		}
	}
	path := "notification_preferences/"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(NotificationPreferences)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
