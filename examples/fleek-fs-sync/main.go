package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	fuse "github.com/FleekHQ/space-poc/examples/fleek-fs-sync/libfuse"
)

// DefaultMountPoint if no mount path is provided
const DefaultMountPoint = "~/.fleek-store"

// This program accepts a mount point as input argument or else defaults to ~/.fleek-store
// - It should mount that directory virtually
// - It should be able to determine when a user adds a new file to the point
// - It should be able to modify the new file added
// - It should be able to determine when a user edits a file
// - It should be able to modify the edited file
// - It should know when a file is deleted
// - It should be able to add a file programmatically
func main() {
	flag.Parse()

	mountPoint := DefaultMountPoint
	if flag.NArg() > 0 {
		mountPoint = flag.Arg(0)
	}

	log.Printf("Mounting at %s\n", mountPoint)
	mirrorPath := "/Users/perfect/Terminal/mirror-path"

	vfs := fuse.NewVFileSystem(mountPoint, mirrorPath)
	if err := vfs.Mount(); err != nil {
		log.Fatal(err)
		return
	}

	// listen for system interrupt
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Printf("Received OS Signal %s", sig.String())
		if err := vfs.Unmount(); err != nil {
			log.Printf("Error Unmounting fuse connection: %s", err.Error())
		}
	}()

	vfs.Serve()
}
