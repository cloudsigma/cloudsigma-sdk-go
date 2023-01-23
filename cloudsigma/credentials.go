package cloudsigma

import (
	"fmt"
)

// Credentials is the CloudSigma credentials value for individual credentials
// fields.
type Credentials struct {
	// Source of the credentials.
	Source string

	// The username (email address) used to communicate with CloudSigma API.
	Username string

	// The password used to communicate with CloudSigma API.
	Password string

	// The access token used to communicate with CloudSigma API.
	Token string
}

// A CredentialsProvider is the interface for any component which will provide
// credentials.
type CredentialsProvider interface {
	Retrieve() (Credentials, error)
}

const UsernamePasswordCredentialsName = "UsernamePasswordCredentials"

type UsernamePasswordCredentialsProvider struct {
	Value Credentials
}

func NewUsernamePasswordCredentialsProvider(username, password string) UsernamePasswordCredentialsProvider {
	return UsernamePasswordCredentialsProvider{
		Value: Credentials{
			Source:   UsernamePasswordCredentialsName,
			Username: username,
			Password: password,
		},
	}
}

func (p UsernamePasswordCredentialsProvider) Retrieve() (Credentials, error) {
	v := p.Value

	if v.Username == "" {
		return Credentials{}, fmt.Errorf("username must not be empty")
	}
	if v.Password == "" {
		return Credentials{}, fmt.Errorf("password must not be empty")
	}

	if v.Source == "" {
		v.Source = UsernamePasswordCredentialsName
	}

	return v, nil
}

const TokenCredentialsName = "TokenCredentials"

type TokenCredentialsProvider struct {
	Value Credentials
}

func NewTokenCredentialsProvider(token string) TokenCredentialsProvider {
	return TokenCredentialsProvider{
		Value: Credentials{
			Source: TokenCredentialsName,
			Token:  token,
		},
	}
}

func (p TokenCredentialsProvider) Retrieve() (Credentials, error) {
	v := p.Value

	if v.Token == "" {
		return Credentials{}, fmt.Errorf("token must not be empty")
	}

	return v, nil
}
