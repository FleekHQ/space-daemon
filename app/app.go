package app

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/FleekHQ/space-poc/core/synchronizers/bucketsync"
	tc "github.com/FleekHQ/space-poc/core/textile/client"
	tl "github.com/FleekHQ/space-poc/core/textile/listener"

	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/store"
	w "github.com/FleekHQ/space-poc/core/watcher"
	"github.com/FleekHQ/space-poc/grpc"
)

// Shutdown logic follows this example https://gist.github.com/akhenakh/38dbfea70dc36964e23acc19777f3869

// Entry point for the app
func Start(ctx context.Context, cfg config.Config) {
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

	// starting the RPC server
	srv := grpc.New(
		store,
		grpc.WithPort(cfg.GetInt(config.SpaceServerPort, 0)),
	)

	g.Go(func() error {
		return srv.Start(ctx)
	})

	watcher, err := w.New(w.WithPaths(cfg.GetString(config.SpaceFolderPath, "")))
	if err != nil {
		log.Fatal(err)
		return
	}

	textileClient := tc.New(store)

	waitForTextileClient := make(chan bool, 1)
	g.Go(func() error {
		if err := textileClient.StartAndBootstrap(); err != nil {
			return err
		}
		waitForTextileClient <- true
		return nil
	})

	<-waitForTextileClient

	// TODO: Iterate over each of the user buckets and create a listener for each one of them
	textileThreadListener := tl.New(textileClient, tc.DefaultPersonalBucketSlug)

	// watcher is started inside bucket sync
	sync := bucketsync.New(watcher, textileClient, textileThreadListener)

	g.Go(func() error {
		return sync.Start(ctx)
	})

	// TODO: add listener services for bucket changes

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
	if sync != nil {
		log.Println("shutdown bucket sync...")
		sync.Stop()
	}

	if srv != nil {
		log.Println("shutdown gRPC server...")
		srv.Stop()
	}

	if store != nil {
		log.Println("shutdown store...")
		store.Close()
	}

	if textileClient != nil {
		log.Println("shutdown Textile client")
		textileClient.Stop()
	}

	if textileThreadListener != nil {
		log.Println("shutdown Textile thread listener")
		textileThreadListener.Close()
	}

	log.Println("waiting for shutdown group")
	err = g.Wait()
	log.Println("finished shutdown group")
	if err != nil {
		log.Println("server returning an error", "error", err)
		os.Exit(2)
	}
}
