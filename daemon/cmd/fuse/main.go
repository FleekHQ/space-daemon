package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/FleekHQ/space/daemon/core/fs_data_source"

	fuse "github.com/FleekHQ/space/daemon/core/libfuse"
	"github.com/FleekHQ/space/daemon/core/spacefs"
)

// DefaultMountPoint if no mount path is provided
const DefaultMountPoint = "/FleekSpace"

// This program accepts a mount point as input argument or else defaults to ~/.fleek-store
// - [x] It should mount that directory virtually
// - [ ] It should be able to determine when a user adds a new file to the point
// - [ ] It should be able to modify the new file added
// - [ ] It should be able to determine when a user edits a file
// - [ ] It should be able to modify the edited file
// - [ ] It should know when a file is deleted
// - [ ] It should be able to add a file programmatically
func main() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
		return
	}
	mountPoint := flag.String("mount", userHomeDir+"/FleekSpace", "Directory on filesystem to mount SpaceFS")
	flag.Parse()

	ctx, cancelCtx := context.WithCancel(context.Background())
	store, err := fs_data_source.NewIpfsDataSource(ctx)
	if err != nil {
		log.Fatal(err)
		return
	}

	sfs, err := spacefs.NewSpaceFS(ctx, store)
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Printf("Mounting at %s\n", *mountPoint)
	vfs := fuse.NewVFileSystem(ctx, *mountPoint, sfs)
	if err := vfs.Mount("FleekSpace"); err != nil {
		log.Fatal(err)
		return
	}

	// listen for system interrupt
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Printf("Received OS Signal %s", sig.String())
		cancelCtx()
		if err := vfs.Unmount(); err != nil {
			log.Printf("Error Unmounting fuse connection: %s", err.Error())
		}
	}()

	err = vfs.Serve()
	if err != nil {
		log.Fatal(err)
	}
}
