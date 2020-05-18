package main

import (
	"flag"
	"log"

	fs "github.com/FleekHQ/space-poc/examples/fleek-fs-sync/filesystem"
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
	vfs := fs.NewVFileSystem(mountPoint)
	err := vfs.Mount()
	if err != nil {
		log.Fatal(err)
	}
}
