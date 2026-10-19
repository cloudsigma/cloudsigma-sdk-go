package cloudsigma

import (
	"context"
	"net/http"
)

// UpdateMailingList edits the profile's mailing_list boolean, including false.
// The legacy ProfileUpdateRequest omits false; this additive method leaves its
// encoding unchanged. Only mailing_list is sent, without echoing readonly
// profile fields. The returned profile is a response snapshot, not proof of
// mailing delivery or subscription processing. Shared Client.Do status and
// response decoding policies apply unchanged.
//
// CloudSigma API docs: https://docs.cloudsigma.com/en/latest/profile.html
func (s *ProfileService) UpdateMailingList(ctx context.Context, enabled bool) (*Profile, *Response, error) {
	body := struct {
		MailingList bool `json:"mailing_list"`
	}{enabled}
	req, err := s.client.NewRequest(http.MethodPut, "profile/", body)
	if err != nil {
		return nil, nil, err
	}
	profile := new(Profile)
	resp, err := s.client.Do(ctx, req, profile)
	if err != nil {
		return nil, resp, err
	}
	return profile, resp, nil
}
