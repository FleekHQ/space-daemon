package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/FleekHQ/space-poc/core/space/fuse"

	"github.com/FleekHQ/space-poc/core/fsds"

	"github.com/FleekHQ/space-poc/core/spacefs"
	"github.com/FleekHQ/space-poc/core/textile"

	"github.com/FleekHQ/space-poc/core/env"
	"github.com/FleekHQ/space-poc/core/space"

	"github.com/FleekHQ/space-poc/core/sync"

	"golang.org/x/sync/errgroup"

	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/store"
	w "github.com/FleekHQ/space-poc/core/watcher"
	"github.com/FleekHQ/space-poc/grpc"
)

// Shutdown logic follows this example https://gist.github.com/akhenakh/38dbfea70dc36964e23acc19777f3869

// Entry point for the app
func Start(ctx context.Context, cfg config.Config, env env.SpaceEnv) {
	// setup to detect interruption
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	g, ctx := errgroup.WithContext(ctx)

	// init store
	store := store.New(
		store.WithPath(cfg.GetString(config.SpaceStorePath, "")),
	)

	waitForStore := make(chan bool, 1)
	g.Go(func() error {
		if err := store.Open(); err != nil {
			return err
		}
		waitForStore <- true
		return nil
	})

	<-waitForStore

	watcher, err := w.New()
	if err != nil {
		log.Fatal(err)
		return
	}

	// setup local buckets
	buckd := textile.NewBuckd()
	g.Go(func() error {
		err := buckd.Start(ctx)
		return err
	})
	<-buckd.WaitForReady()

	// setup textile client
	bootstrapReady := make(chan bool)
	textileClient := textile.NewClient(store)
	g.Go(func() error {
		err := textileClient.StartAndBootstrap(ctx, cfg)
		bootstrapReady <- true
		return err
	})

	// wait for textileClient to initialize
	<-textileClient.WaitForReady()
	<-bootstrapReady

	// watcher is started inside bucket sync
	sync := sync.New(watcher, textileClient, store, nil)

	// setup the RPC server and Service
	sv, svErr := space.NewService(
		store,
		textileClient,
		sync,
		cfg,
		space.WithEnv(env),
	)

	// setup FUSE FS Handler
	sfs, err := spacefs.New(ctx, fsds.NewSpaceFSDataSource(sv))
	if err != nil {
		log.Fatal(err)
	}
	fuseController := fuse.NewController(ctx, cfg, store, sfs)
	if fuseController.ShouldMount() {
		log.Println("Mounting FUSE Drive")
		if err := fuseController.Mount(); err != nil {
			log.Println(errors.Wrap(err, "could not mount fuse on startup"))
		}
	}

	srv := grpc.New(
		sv,
		fuseController,
		grpc.WithPort(cfg.GetInt(config.SpaceServerPort, 0)),
	)

	g.Go(func() error {
		sync.RegisterNotifier(srv)
		return sync.Start(ctx)
	})

	<-sync.WaitForReady()

	// start the gRPC server
	g.Go(func() error {
		if svErr != nil {
			log.Printf("unable to initialize service %s\n", svErr.Error())
			return svErr
		}
		return srv.Start(ctx)
	})

	fmt.Println("daemon ready")

	// wait for interruption or done signal
	select {
	case <-interrupt:
		log.Println("got interrupt signal")
		break
	case <-ctx.Done():
		log.Println("got context done signal")
		break
	}

	// shutdown gracefully
	log.Println("received shutdown signal. ")

	_, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// probably we can create an interface Stop/Close to loop thru all modules
	// NOTE: need to make sure the order of shutdown is in sync and we dont drop events
	if textileClient != nil {
		log.Println("shutdown Textile client")
		textileClient.Stop()
	}

	if sync != nil {
		log.Println("shutdown bucket sync...")
		sync.Stop()
	}

	err = fuseController.Unmount()
	if err != nil {
		log.Println(errors.Wrap(err, "error shutdown FUSE Drive"))
	}

	if srv != nil {
		log.Println("shutdown gRPC server...")
		srv.Stop()
	}

	if store != nil {
		log.Println("shutdown store...")
		store.Close()
	}

	if buckd != nil {
		log.Println("shutdown Buckd node")
		buckd.Stop()
	}

	log.Println("waiting for shutdown group")
	err = g.Wait()
	log.Println("finished shutdown group")
	if err != nil {
		log.Println("server returning an error", "error", err)
		os.Exit(2)
	}
}
