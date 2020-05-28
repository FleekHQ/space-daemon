package ipfs

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	blockservice "github.com/ipfs/go-blockservice"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreunix"
	ipld "github.com/ipfs/go-ipld-format"
	dag "github.com/ipfs/go-merkledag"
	dagtest "github.com/ipfs/go-merkledag/test"
	mfs "github.com/ipfs/go-mfs"
	ft "github.com/ipfs/go-unixfs"
	path "github.com/ipfs/interface-go-ipfs-core/path"
)

var nilNode *core.IpfsNode
var once sync.Once

type syncDagService struct {
	ipld.DAGService
	syncFn func() error
}

func getOrCreateNilNode() (*core.IpfsNode, error) {
	once.Do(func() {
		if nilNode != nil {
			return
		}
		node, err := core.NewNode(context.Background(), &core.BuildCfg{
			NilRepo: true,
		})
		if err != nil {
			panic(err)
		}
		nilNode = node
	})

	return nilNode, nil
}

// GetFileHash returns the hash of a given file without uploading it (sha2-256)
// Stripped down version of this so we can calc hash without running a full node
// https://github.com/ipfs/go-ipfs/blob/master/core/coreapi/unixfs.go#L57
func GetFileHash(r io.Reader) (*string, error) {
	node, err := getOrCreateNilNode()
	if err != nil {
		return nil, err
	}
	addblockstore := node.Blockstore
	exch := node.Exchange
	pinning := node.Pinning

	bserv := blockservice.New(addblockstore, exch) // hash security 001
	dserv := dag.NewDAGService(bserv)

	syncDserv := &syncDagService{
		DAGService: dserv,
		syncFn:     func() error { return nil },
	}

	ctx := context.Background()

	fileAdder, err := coreunix.NewAdder(ctx, pinning, addblockstore, syncDserv)
	if err != nil {
		return nil, err
	}

	prefix, err := dag.PrefixForCidVersion(1)
	if err != nil {
		return nil, fmt.Errorf("bad CID Version: %s", err)
	}

	fileAdder.CidBuilder = prefix

	file := files.NewReaderFile(r)

	md := dagtest.Mock()
	emptyDirNode := ft.EmptyDirNode()
	emptyDirNode.SetCidBuilder(fileAdder.CidBuilder)
	mr, err := mfs.NewRoot(ctx, md, emptyDirNode, nil)
	if err != nil {
		return nil, err
	}

	fileAdder.SetMfsRoot(mr)

	nd, err := fileAdder.AddAllAndPin(file)
	if err != nil {
		return nil, err
	}

	cid := strings.Split(path.IpfsPath(nd.Cid()).String(), "/")[2]
	return &cid, nil
}
