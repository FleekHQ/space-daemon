// +build linux darwin

package fuse

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"

	"github.com/FleekHQ/space-daemon/core/spacefs"
	"github.com/FleekHQ/space-daemon/mocks"
	fusemocks "github.com/FleekHQ/space-daemon/mocks/fuse"
)

type testCtx struct {
	cfg       *mocks.Config
	st        *mocks.Store
	fsds      *fusemocks.FSDataSource
	installer *fusemocks.FuseInstaller
}

func initTestCtx() (context.Context, *testCtx, *Controller) {
	tctx := &testCtx{
		cfg:       new(mocks.Config),
		st:        new(mocks.Store),
		fsds:      new(fusemocks.FSDataSource),
		installer: new(fusemocks.FuseInstaller),
	}

	ctx := context.Background()
	fs := spacefs.New(tctx.fsds)

	controller := NewController(ctx, tctx.cfg, tctx.st, fs, tctx.installer)
	return ctx, tctx, controller
}

func TestController_GetFuseState_ShouldDefaultTo_Not_Installed(t *testing.T) {
	ctx, test, controller := initTestCtx()

	test.installer.On("IsInstalled", mock.Anything).Return(false, nil)

	state, err := controller.GetFuseState(ctx)
	assert.NoError(t, err, "error on GetFuseState()")

	assert.Equal(t, NOT_INSTALLED, state, "unexpected state gotten")
}

// Note: This is more of an integration test than unit test, but should run cleanly across multiple threads
func TestController_GetFuseState_ShouldBe_Unmounted_When_Installed(t *testing.T) {
	ctx, test, controller := initTestCtx()

	test.installer.On("IsInstalled", mock.Anything).Return(true, nil)

	state, err := controller.GetFuseState(ctx)
	assert.NoError(t, err, "error on GetFuseState()")

	assert.Equal(t, UNMOUNTED, state, "unexpected state gotten")
}
