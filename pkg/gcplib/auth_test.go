package gcplib

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func DisableTest_TestGetCurrentUserAccessToken(t *testing.T) {
	token, err := GetCurrentUserAccessToken()
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
}
