package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

// KeypairGenerateRequest requests a server-generated SSH keypair. AlgorithmType
// must be "rsa" or "ed25519". No key material is supplied by this request.
// This separate type leaves legacy Keypair Create/Update serialization unchanged.
type KeypairGenerateRequest struct {
	Name          string `json:"name,omitempty"`
	AlgorithmType string `json:"algorithm_type"`
}

// Generate requests one new keypair using the explicitly selected algorithm.
// Redirects are refused with ErrKeypairGenerateRedirect. The initial POST
// outcome is unknown; callers must not blindly retry.
// The response may contain PRIVATE KEY MATERIAL; callers must handle it securely.
// This uses the ordinary keypairs authentication path (including any API OTP
// requirements); it does not bypass two-factor authentication. A response is not
// proof that a key was installed on a server. Invalid algorithms are rejected
// locally, a deliberate SDK policy based on the documented supported choices.
//
// CloudSigma API docs: https://docs.cloudsigma.com/en/latest/keypairs.html#creating-a-keypair
func (s *KeypairsService) Generate(ctx context.Context, input *KeypairGenerateRequest) ([]Keypair, *Response, error) {
	if input == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}
	if input.AlgorithmType != "rsa" && input.AlgorithmType != "ed25519" {
		return nil, nil, fmt.Errorf("cloudsigma: keypair algorithm must be rsa or ed25519")
	}
	payload := struct {
		Objects []KeypairGenerateRequest `json:"objects"`
	}{Objects: []KeypairGenerateRequest{*input}}
	req, err := s.client.NewRequest(http.MethodPost, keypairsBasePath+"/", payload)
	if err != nil {
		return nil, nil, err
	}
	root := new(KeypairCreateRequest)
	resp, err := s.client.doResourceAction(ctx, req, root, ErrKeypairGenerateRedirect)
	if err != nil {
		return nil, resp, err
	}
	return root.Keypairs, resp, nil
}
