package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/FleekHQ/space-daemon/core"

	"github.com/FleekHQ/space-daemon/core/space/fuse"

	"github.com/FleekHQ/space-daemon/core/fsds"

	"github.com/FleekHQ/space-daemon/core/spacefs"
	textile "github.com/FleekHQ/space-daemon/core/textile"

	"github.com/FleekHQ/space-daemon/core/env"
	"github.com/FleekHQ/space-daemon/core/space"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/sync"
	"github.com/FleekHQ/space-daemon/log"

	node "github.com/FleekHQ/space-daemon/core/ipfs/node"

	"golang.org/x/sync/errgroup"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/store"
	w "github.com/FleekHQ/space-daemon/core/watcher"
	"github.com/FleekHQ/space-daemon/grpc"
	"github.com/golang-collections/collections/stack"
)

// Shutdown logic follows this example https://gist.github.com/akhenakh/38dbfea70dc36964e23acc19777f3869
type App struct {
	eg             *errgroup.Group
	components     *stack.Stack
	cfg            config.Config
	env            env.SpaceEnv
	isShuttingDown bool
}

type componentMap struct {
	name      string
	component core.Component
}

func New(cfg config.Config, env env.SpaceEnv) *App {
	return &App{
		components:     stack.New(),
		cfg:            cfg,
		env:            env,
		isShuttingDown: false,
	}
}

// Start is the Entry point for the app.
// All module components are initialized and managed here.
// When a top level module that need to be shutdown on exit is initialized. It should be
// added to the apps list of tracked components using the `Run()` function, but if the component has a blocking
// start/run function it should be tracked with the `RunAsync()` function and call the blocking function in the
// input function block.
func (a *App) Start(ctx context.Context) error {
	a.eg, ctx = errgroup.WithContext(ctx)

	// setup to detect interruption
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	// init appStore
	appStore := store.New(
		store.WithPath(a.cfg.GetString(config.SpaceStorePath, "")),
	)
	if err := appStore.Open(); err != nil {
		return err
	}
	a.Run("Store", appStore)

	watcher, err := w.New()
	if err != nil {
		return err
	}
	a.Run("FolderWatcher", watcher)

	// setup local ipfs node
	node := node.NewIpsNode(a.cfg)
	a.RunAsync("IpfsNode", node, func() error {
		return node.Start(ctx)
	})

	// setup local buckets
	buckd := textile.NewBuckd(a.cfg)
	a.RunAsync("BucketDaemon", buckd, func() error {
		return buckd.Start(ctx)
	})

	// setup textile client
	textileClient := textile.NewClient(appStore)
	a.RunAsync("TextileClient", textileClient, func() error {
		return textileClient.Start(ctx, a.cfg)
	})

	// watcher is started inside bucket sync
	bucketSync := sync.New(watcher, textileClient, appStore, nil)

	// setup the Space Service
	sv, svErr := space.NewService(
		appStore,
		textileClient,
		bucketSync,
		a.cfg,
		keychain.New(appStore),
		space.WithEnv(a.env),
	)
	if svErr != nil {
		return svErr
	}

	// setup FUSE FS Handler
	sfs, err := spacefs.New(fsds.NewSpaceFSDataSource(sv))
	if err != nil {
		log.Error("Failed to create space FUSE data source", err)
		return err
	}
	fuseController := fuse.NewController(ctx, a.cfg, appStore, sfs)
	if fuseController.ShouldMount() {
		log.Info("Mounting FUSE Drive")
		if err := fuseController.Mount(); err != nil {
			log.Error("Mounting FUSE drive failed", err)
		} else {
			log.Info("Mounting FUSE Drive successful")
		}
	}
	a.Run("FuseController", fuseController)

	// setup gRPC Server
	srv := grpc.New(
		sv,
		fuseController,
		grpc.WithPort(a.cfg.GetInt(config.SpaceServerPort, 0)),
		grpc.WithProxyPort(a.cfg.GetInt(config.SpaceProxyServerPort, 0)),
		grpc.WithRestProxyPort(a.cfg.GetInt(config.SpaceRestProxyServerPort, 0)),
	)

	a.RunAsync("BucketSync", bucketSync, func() error {
		bucketSync.RegisterNotifier(srv)
		return bucketSync.Start(ctx)
	})

	// start the gRPC server
	a.RunAsync("gRPCServer", srv, func() error {
		return srv.Start(ctx)
	})

	log.Info("Daemon ready")

	// wait for interruption or done signal
	select {
	case <-interrupt:
		log.Debug("Got interrupt signal")
		// TODO: Track multiple interrupts in a goroutine to force exit for app.
		break
	case <-ctx.Done():
		log.Debug("Got context done signal")
		break
	}

	return a.Shutdown()
}

// Run registers this component to be cleaned up on Shutdown
func (a *App) Run(name string, component core.Component) {
	log.Debug("Starting Component", "name:"+name)
	a.components.Push(&componentMap{
		name:      name,
		component: component,
	})
}

// RunAsync performs the same function as Run() but also accepts an function to be run
// async to initialize the component.
func (a *App) RunAsync(name string, component core.AsyncComponent, fn func() error) {
	log.Debug("Starting Async Component", "name:"+name)
	if a.eg == nil {
		log.Warn("App.RunAsync() should be called after App.Start()")
		return
	}

	a.eg.Go(func() error {
		return fn()
	})

	<-component.WaitForReady()
	a.components.Push(&componentMap{
		name:      name,
		component: component,
	})
}

// Shutdown would perform a graceful shutdown of all components added through the
// Run() or RunAsync() functions
func (a *App) Shutdown() error {
	log.Info("Daemon shutdown started")
	for a.components.Len() > 0 {
		m, ok := a.components.Pop().(*componentMap)
		if ok {
			log.Debug("Shutting down Component", fmt.Sprintf("name:%s", m.name))
			if err := m.component.Shutdown(); err != nil {
				log.Error(fmt.Sprintf("Error shutting down %s", m.name), err)
			}
		}
	}

	err := a.eg.Wait()
	log.Info("Shutdown complete")
	return err
}
