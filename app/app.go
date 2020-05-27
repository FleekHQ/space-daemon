package app

import (
	"context"
	"golang.org/x/sync/errgroup"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FleekHQ/space-poc/core/synchronizers/bucketsync"
	tc "github.com/FleekHQ/space-poc/core/textile/client"

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

	g.Go(func() error {
		return store.Open()
	})

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

	// Testing bucket creation here
	if err := textileClient.CreateBucket("my-bucket"); err != nil {
		log.Fatal("error creating bucket", err)
	} else {
		log.Printf("Created bucket successfully")
	}

	// watcher is started inside bucket sync
	sync := bucketsync.New(watcher, textileClient)

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

	log.Println("waiting for shutdown group")
	err = g.Wait()
	log.Println("finished shutdown group")
	if err != nil {
		log.Println("server returning an error", "error", err)
		os.Exit(2)
	}
}
