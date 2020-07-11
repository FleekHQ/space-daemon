package fuse

// VFS represents the handler for virtually mounted drives.
// it is implemented using FUSE for linux and macOS
// and will use dokany for windows
type VFS interface {
	Mount(mountPath, fsName string) error
	IsMounted() bool
	// Serve should be a blocking call and return only on unmount or shutdown
	Serve() error
	Unmount() error
}
