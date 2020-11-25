package installer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// NOTE: This is more of an integration test and is commented out by default till functional test-suite is ready
func TestMacFuseInstaller(t *testing.T) {
	ctx := context.Background()
	installer := NewFuseInstaller()

	installed, err := installer.IsInstalled(ctx)
	assert.NoError(t, err)
	assert.Equal(t, false, installed, "fuse should not be installed by default")
}
