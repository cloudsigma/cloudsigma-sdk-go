package cloudsigma

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsernamePasswordCredentials_Retrieve(t *testing.T) {
	credProvider := NewUsernamePasswordCredentialsProvider("username", "password")

	cred, err := credProvider.Retrieve()

	assert.NoError(t, err)
	assert.Equal(t, UsernamePasswordCredentialsName, cred.Source)
	assert.Equal(t, "username", cred.Username)
	assert.Equal(t, "password", cred.Password)
}

func TestUsernamePasswordCredentials_Retrieve_emptyUsername(t *testing.T) {
	credProvider := NewUsernamePasswordCredentialsProvider("", "password")

	_, err := credProvider.Retrieve()

	assert.Error(t, err)
}

func TestUsernamePasswordCredentials_Retrieve_emptyPassword(t *testing.T) {
	credProvider := NewUsernamePasswordCredentialsProvider("username", "")

	_, err := credProvider.Retrieve()

	assert.Error(t, err)
}

func TestUsernamePasswordCredentials_Retrieve_fixMissingSource(t *testing.T) {
	credProvider := NewUsernamePasswordCredentialsProvider("username", "password")
	credProvider.Value.Source = ""

	cred, err := credProvider.Retrieve()

	assert.NoError(t, err)
	assert.Equal(t, UsernamePasswordCredentialsName, cred.Source)
}

func TestTokenCredentials_Retrieve(t *testing.T) {
	credProvider := NewTokenCredentialsProvider("token")

	cred, err := credProvider.Retrieve()

	assert.NoError(t, err)
	assert.Equal(t, TokenCredentialsName, cred.Source)
	assert.Equal(t, "token", cred.Token)
}

func TestTokenCredentials_Retrieve_emptyToken(t *testing.T) {
	credProvider := NewTokenCredentialsProvider("")

	_, err := credProvider.Retrieve()

	assert.Error(t, err)
}
