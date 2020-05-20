package spacefs

import (
	"context"
	"github.com/FleekHQ/space-poc/examples/fleek-fs-sync/spacestore"
	format "github.com/ipfs/go-ipld-format"
	"log"
	"syscall"

	cid "github.com/ipfs/go-cid"
)

// SpaceFS is represents the filesystem
// It implements the FSOps interface
// And is responsible for managing file access, encryption and decryption
type SpaceFS struct {
	ctx       context.Context
	store     spacestore.SpaceStore
}

// NewSpaceFS initializes a SpaceFS instance with IPFS peer listening
func NewSpaceFS(ctx context.Context, store spacestore.SpaceStore) (*SpaceFS, error) {
	return &SpaceFS{
		ctx:       ctx,
		store:     store,
	}, nil
}

// Root implements the FSOps Root function
// It returns the root directory of the file
func (fs *SpaceFS) Root() (DirEntryOps, error) {
	// TODO: fetch the root block node and cache their information locally
	return &SpaceDirectory{
		path: "/",
	}, nil
}

// LookupPath implements the FsOps interface for looking up information
// in a directory
func (fs *SpaceFS) LookupPath(path string) (DirEntryOps, error) {
	log.Printf("Getting item at path: %s", path)
	node, err := fs.store.Get(fs.ctx, path)

	if err != nil {
		return nil, syscall.ENOENT
	}

	stats, err := node.Stat()
	if err != nil {
		return nil, syscall.ENOENT
	}

	for _, link := range node.Links() {
		log.Printf("Link %+v", link)
	}

	childNode, remaining, err := node.Resolve([]string{"static"})
	log.Printf("%s Child Node: %+v", "static", childNode)
	log.Printf("Remaining %+v", remaining)
	log.Printf("Error: %+v", err)

	return &SpaceDirectory{
		cid: node.Cid(),
		node: node,
		name: stats.String(),
		path: path,
	}, nil
}

// SpaceDirectory is a directory managed by space
type SpaceDirectory struct {
	cid   cid.Cid
	node format.Node
	name string
	path  string
}

func (dir *SpaceDirectory) Cid() cid.Cid {
	return dir.cid
}

// Path implements DirEntryOps Path() and return the path of the directory
func (dir *SpaceDirectory) Path() string {
	return dir.path
}

// Attribute implements DirEntryOps Attribute() and fetches the metadata of the directory
func (dir *SpaceDirectory) Attribute() (DirEntryAttribute, error) {
	return nil, nil
}

// ReadDir implements DirOps ReadDir and returns the list of entries in a directory
func (dir *SpaceDirectory) ReadDir() ([]DirEntryOps, error) {
	return nil, nil
}

// SpaceFile is a file managed by space
type SpaceFile struct {
	Parent *SpaceDirectory
	cid    cid.Cid
	path   string
}

func (f *SpaceFile) Cid() cid.Cid {
	return f.cid
}

// Path implements DirEntryOps Path() and return the path of the directory
func (f *SpaceFile) Path() string {
	return f.path
}

// Attribute implements DirEntryOps Attribute() and fetches the metadata of the directory
func (f *SpaceFile) Attribute() (DirEntryAttribute, error) {
	return nil, nil
}

// Open implements FileOps Open
// It should download/cache the content of the file and return a fileHandler
// that can read the cached content.
func (f *SpaceFile) Open() (FileHandler, error) {
	return nil, nil
}
