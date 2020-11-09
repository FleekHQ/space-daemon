package permissions_test

import (
	"testing"

	"github.com/FleekHQ/space-daemon/core/permissions"
	"github.com/stretchr/testify/assert"
)

func TestPermissions_AppToken_Generation(t *testing.T) {
	tok, err := permissions.GenerateRandomToken(true, []string{})
	assert.NoError(t, err)

	marshalled := permissions.MarshalFullToken(tok)
	unmarshalled, err := permissions.UnmarshalFullToken(marshalled)
	assert.NoError(t, err)

	assert.Equal(t, tok, unmarshalled)
}

func TestPermissions_AppToken_GenerationWithPerms(t *testing.T) {
	tok, err := permissions.GenerateRandomToken(false, []string{"OpenFile", "ListDirectories"})
	assert.NoError(t, err)

	marshalled := permissions.MarshalFullToken(tok)
	unmarshalled, err := permissions.UnmarshalFullToken(marshalled)
	assert.NoError(t, err)

	assert.Equal(t, tok, unmarshalled)
}
