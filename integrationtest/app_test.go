package integrationtest

import (
	"context"
	"runtime"
	"testing"

	"github.com/FleekHQ/space-daemon/config"
	spaceEnv "github.com/FleekHQ/space-daemon/core/env"

	"github.com/stretchr/testify/assert"

	spaceApp "github.com/FleekHQ/space-daemon/app"
)

func TestAppDoesNotLeakGoroutines(t *testing.T) {
	goRoutinesBefore := runtime.NumGoroutine()

	cf := &config.Flags{}
	// env
	env := spaceEnv.New()

	// load configs
	cfg := config.NewMap(env, cf)
	ctx, cancelCtx := context.WithCancel(context.Background())
	app := spaceApp.New(cfg, env)

	var err error
	errChan := make(chan error)
	go func() {
		errChan <- app.Start(ctx)
	}()

	select {
	case <-app.WaitForReady():
	case err = <-errChan:
	}

	assert.Nil(t, err, "app.Start() Failed")
	cancelCtx()
	err = <-errChan
	assert.Nil(t, err, "app.Shutdown() Failed")

	assert.Equal(t, goRoutinesBefore, runtime.NumGoroutine(), "Goroutine leaked on app Shutdown")
}
