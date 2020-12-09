package app

import (
	"context"
	"fmt"

	"github.com/FleekHQ/space-daemon/core/space/fuse/installer"

	"github.com/FleekHQ/space-daemon/core/search/bleve"
	"github.com/pkg/errors"

	"github.com/FleekHQ/space-daemon/core"
	"github.com/FleekHQ/space-daemon/grpc"

	"github.com/FleekHQ/space-daemon/core/space/fuse"
	"github.com/FleekHQ/space-daemon/core/vault"

	"github.com/FleekHQ/space-daemon/core/fsds"

	"github.com/FleekHQ/space-daemon/core/spacefs"
	textile "github.com/FleekHQ/space-daemon/core/textile"
	"github.com/FleekHQ/space-daemon/core/textile/hub"

	"github.com/FleekHQ/space-daemon/core/env"
	"github.com/FleekHQ/space-daemon/core/space"

	node "github.com/FleekHQ/space-daemon/core/ipfs/node"
	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/sync"
	"github.com/FleekHQ/space-daemon/log"

	"golang.org/x/sync/errgroup"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/store"
	w "github.com/FleekHQ/space-daemon/core/watcher"
	"github.com/golang-collections/collections/stack"
)

// Shutdown logic follows this example https://gist.github.com/akhenakh/38dbfea70dc36964e23acc19777f3869
type App struct {
	eg         *errgroup.Group
	components *stack.Stack
	cfg        config.Config
	env        env.SpaceEnv
	IsRunning  bool
}

type componentMap struct {
	name      string
	component core.Component
}

func New(cfg config.Config, env env.SpaceEnv) *App {
	return &App{
		components: stack.New(),
		cfg:        cfg,
		env:        env,
		IsRunning:  false,
	}
}

// Start is the Entry point for the app.
// All module components are initialized and managed here.
// When a top level module that need to be shutdown on exit is initialized. It should be
// added to the apps list of tracked components using the `Run()` function, but if the component has a blocking
// start/run function it should be tracked with the `RunAsync()` function and call the blocking function in the
// input function block.
func (a *App) Start() error {
	var ctx context.Context
	a.eg, ctx = errgroup.WithContext(context.Background())

	log.SetLogLevel(a.cfg.GetString(config.LogLevel, "debug"))

	// init appStore
	appStore := store.New(
		store.WithPath(a.cfg.GetString(config.SpaceStorePath, "")),
	)
	if err := appStore.Open(); err != nil {
		return err
	}
	a.Run("Store", appStore)

	// Init keychain
	kc := keychain.New(keychain.WithPath(a.cfg.GetString(config.SpaceStorePath, "")), keychain.WithStore(appStore))

	// Init Vault
	v := vault.New(a.cfg.GetString(config.SpaceVaultAPIURL, ""), a.cfg.GetString(config.SpaceVaultSaltSecret, ""))

	watcher, err := w.New()
	if err != nil {
		return err
	}
	a.Run("FolderWatcher", watcher)

	// setup local ipfs node if Ipfsnode is set
	if a.cfg.GetBool(config.Ipfsnode, true) {
		// setup local ipfs node
		node := node.NewIpsNode(a.cfg)
		err = a.RunAsync("IpfsNode", node, func() error {
			return node.Start(ctx)
		})
		if err != nil {
			log.Error("error starting embedded IPFS node", err)
			return err
		}
	} else {
		log.Info("Skipping embedded IPFS node")
	}

	// setup local buckets
	buckd := textile.NewBuckd(a.cfg)
	err = a.RunAsync("BucketDaemon", buckd, func() error {
		return buckd.Start(ctx)
	})
	if err != nil {
		return err
	}

	hubAuth := hub.New(appStore, kc, a.cfg)

	// setup files search engine
	searchEngine := bleve.NewSearchEngine(bleve.WithDBPath(a.cfg.GetString(config.SpaceStorePath, "")))
	a.Run("FilesSearchEngine", searchEngine)

	// setup textile client
	uc := textile.CreateUserClient(a.cfg.GetString(config.TextileHubTarget, ""))
	textileClient := textile.NewClient(appStore, kc, hubAuth, uc, nil, searchEngine)
	err = a.RunAsync("TextileClient", textileClient, func() error {
		return textileClient.Start(ctx, a.cfg)
	})
	if err != nil {
		return err
	}

	// watcher is started inside bucket sync
	bucketSync := sync.New(watcher, textileClient, appStore, nil)

	// setup the Space Service
	sv, svErr := space.NewService(
		appStore,
		textileClient,
		bucketSync,
		a.cfg,
		kc,
		v,
		hubAuth,
		space.WithEnv(a.env),
	)
	if svErr != nil {
		return svErr
	}

	// setup FUSE FS Handler
	sfs := spacefs.New(fsds.NewSpaceFSDataSource(
		sv,
		fsds.WithFilesDataSources(sv),
		fsds.WithSharedWithMeDataSources(sv),
	))
	fuseInstaller := installer.NewFuseInstaller()
	fuseController := fuse.NewController(ctx, a.cfg, appStore, sfs, fuseInstaller)
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
		kc,
		grpc.WithPort(a.cfg.GetInt(config.SpaceServerPort, 0)),
		grpc.WithProxyPort(a.cfg.GetInt(config.SpaceProxyServerPort, 0)),
		grpc.WithRestProxyPort(a.cfg.GetInt(config.SpaceRestProxyServerPort, 0)),
	)

	textileClient.AttachMailboxNotifier(srv)
	textileClient.AttachSynchronizerNotifier(srv)

	// start the gRPC server
	err = a.RunAsync("gRPCServer", srv, func() error {
		return srv.Start(ctx)
	})
	if err != nil {
		return err
	}

	err = a.RunAsync("BucketSync", bucketSync, func() error {
		bucketSync.RegisterNotifier(srv)
		return bucketSync.Start(ctx)
	})
	if err != nil {
		return err
	}

	log.Info("Daemon ready")
	a.IsRunning = true

	return nil
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
func (a *App) RunAsync(name string, component core.AsyncComponent, fn func() error) error {
	log.Debug("Starting Async Component", "name:"+name)
	if a.eg == nil {
		log.Warn("App.RunAsync() should be called after App.Start()")
		return nil
	}

	errc := make(chan error)

	a.eg.Go(func() error {
		err := fn()
		if err != nil {
			errc <- err
		}

		return err
	})

	select {
	case err := <-errc:
		return err
	case <-component.WaitForReady():
		a.components.Push(&componentMap{
			name:      name,
			component: component,
		})
	}

	return nil
}

// Shutdown would perform a graceful shutdown of all components added through the
// Run() or RunAsync() functions
func (a *App) Shutdown() error {
	log.Info("Daemon shutdown started")
	if !a.IsRunning {
		return errors.New("app is not running")
	}

	for a.components.Len() > 0 {
		m, ok := a.components.Pop().(*componentMap)
		if ok {
			log.Debug("Shutting down Component", fmt.Sprintf("name:%s", m.name))
			if err := m.component.Shutdown(); err != nil {
				log.Error(fmt.Sprintf("error shutting down %s", m.name), err)
			}
		}
	}

	err := a.eg.Wait()
	log.Info("Shutdown complete")
	a.IsRunning = false
	return err
}
