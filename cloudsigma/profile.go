package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const profileBasePath = "profile"

// ProfileService handles communication with the profile related methods of the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/profile.html
type ProfileService service

// Profile represents a CloudSigma user profile.
type Profile struct {
	Address              string                 `json:"address,omitempty"`
	APIHTTPSOnly         bool                   `json:"api_https_only,omitempty"`
	AutoTopUpAmount      string                 `json:"autotopup_amount,omitempty"`
	AutoTopUpThreshold   string                 `json:"autotopup_threshold,omitempty"`
	BankReference        string                 `json:"bank_reference,omitempty"`
	CloneNaming          string                 `json:"clone_naming,omitempty"`
	Company              string                 `json:"company,omitempty"`
	Country              string                 `json:"country,omitempty"`
	Currency             string                 `json:"currency,omitempty"`
	Email                string                 `json:"email,omitempty"`
	FirstName            string                 `json:"first_name,omitempty"`
	HasAutoTopUp         bool                   `json:"has_autotopup,omitempty"`
	HasTxAutoTopUp       bool                   `json:"has_tx_autotopup,omitempty"`
	Invoicing            bool                   `json:"invoicing,omitempty"`
	KeyAuth              bool                   `json:"key_auth,omitempty"`
	Language             string                 `json:"language,omitempty"`
	LastName             string                 `json:"last_name,omitempty"`
	MailingList          bool                   `json:"mailing_list,omitempty"`
	Meta                 map[string]interface{} `json:"meta,omitempty"`
	MyNotes              string                 `json:"my_notes,omitempty"`
	NetworkRestrictions  string                 `json:"network_restrictions,omitempty"`
	Nickname             string                 `json:"nickname,omitempty"`
	Phone                string                 `json:"phone,omitempty"`
	Postcode             string                 `json:"postcode,omitempty"`
	Reseller             string                 `json:"reseller,omitempty"`
	SignupTime           string                 `json:"signup_time,omitempty"`
	State                string                 `json:"state,omitempty"`
	TaxName              string                 `json:"tax_name,omitempty"`
	TaxRate              string                 `json:"tax_rate,omitempty"`
	Title                string                 `json:"title,omitempty"`
	Town                 string                 `json:"town,omitempty"`
	TxAutoTopUpAmount    string                 `json:"tx_autotopup_amount,omitempty"`
	TxAutoTopUpThreshold string                 `json:"tx_autotopup_threshold,omitempty"`
	UUID                 string                 `json:"uuid"`
	VAT                  string                 `json:"vat,omitempty"`
}

// ProfileUpdateRequest represents a request to update a profile.
type ProfileUpdateRequest struct {
	*Profile
}

// Get provides information for an user profile.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/profile.html#listing
func (s *ProfileService) Get(ctx context.Context) (*Profile, *Response, error) {
	path := fmt.Sprintf("%v/", profileBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
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

// Update edits a user profile.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/profile.html#editing
func (s *ProfileService) Update(ctx context.Context, updateRequest *ProfileUpdateRequest) (*Profile, *Response, error) {
	if updateRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/", profileBasePath)

	req, err := s.client.NewRequest(http.MethodPut, path, updateRequest)
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
