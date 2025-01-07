package crypto

import (
	"github.com/endless-labs/endless-go-sdk/internal/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

const testAuthKey = "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

func TestAuthenticationKey_CryptoMaterial(t *testing.T) {
	authKeyBytes, err := util.ParseHex(testAuthKey)
	assert.NoError(t, err)

	authKeyFromString := &AuthenticationKey{}
	err = authKeyFromString.FromHex(testAuthKey)
	assert.NoError(t, err)

	authKeyFromBytes := &AuthenticationKey{}
	err = authKeyFromBytes.FromBytes(authKeyBytes)
	assert.NoError(t, err)

	assert.Equal(t, authKeyFromString, authKeyFromBytes)

	assert.Equal(t, authKeyBytes, authKeyFromString.Bytes())
	assert.Equal(t, testAuthKey, authKeyFromString.ToHex())

	assert.Equal(t, authKeyBytes, authKeyFromBytes.Bytes())
	assert.Equal(t, testAuthKey, authKeyFromBytes.ToHex())
}

func TestAuthenticationKey_CryptoMaterialError(t *testing.T) {
	authKey := &AuthenticationKey{}
	err := authKey.FromHex("0x123456")
	assert.Error(t, err) // Not long enough

	err = authKey.FromHex("abcde")
	assert.Error(t, err) // Not a string
}
