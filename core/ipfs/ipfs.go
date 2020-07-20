package ipfs

import (
	"context"
	"io"
	"sync"

	"github.com/ipfs/go-cid"

	"github.com/pkg/errors"

	"github.com/ipfs/interface-go-ipfs-core/path"

	"github.com/FleekHQ/space-daemon/log"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/FleekHQ/space-daemon/config"
	files "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
)

type AddItemResult struct {
	Error    error
	Resolved path.Resolved
}

type LinkNodesInput struct {
	// name of link
	Name string
	Path path.Path
}

type LinkNodesResult struct {
	ParentPath path.Resolved
}

type Client interface {
	AddItems(ctx context.Context, items []io.Reader) []AddItemResult
	AddItem(ctx context.Context, item io.Reader) AddItemResult
	// Links each of the nodes in the input under the same parent
	LinkNodes(ctx context.Context, nodes []LinkNodesInput) (*LinkNodesResult, error)
	PullItem(ctx context.Context, cid cid.Cid) (io.ReadCloser, error)
}

type SpaceIpfsClient struct {
	client *httpapi.HttpApi
}

func NewSpaceIpfsClient(cfg config.Config) (*SpaceIpfsClient, error) {
	ipfsAddr := cfg.GetString(config.Ipfsaddr, "/ip4/127.0.0.1/tcp/5001")

	multiaddress, err := ma.NewMultiaddr(ipfsAddr)
	if err != nil {
		log.Error("Unable to parse IPFS Multiaddr", err)
		return nil, err
	}

	ic, err := httpapi.NewApi(multiaddress)
	if err != nil {
		return nil, err
	}
	return &SpaceIpfsClient{
		client: ic,
	}, nil
}

func (s *SpaceIpfsClient) AddItems(ctx context.Context, items []io.Reader) []AddItemResult {
	results := make([]AddItemResult, len(items))
	wg := sync.WaitGroup{}

	for i, item := range items {
		wg.Add(1)
		go func(i int, item io.Reader) {
			resolved, err := s.client.Unixfs().Add(
				ctx,
				files.NewReaderFile(item),
			)
			results[i] = AddItemResult{
				Error:    err,
				Resolved: resolved,
			}
			wg.Done()
		}(i, item)
	}

	wg.Wait()
	return results
}

func (s *SpaceIpfsClient) AddItem(ctx context.Context, item io.Reader) AddItemResult {
	result := s.AddItems(ctx, []io.Reader{item})
	return result[0]
}

func (s *SpaceIpfsClient) LinkNodes(ctx context.Context, nodes []LinkNodesInput) (*LinkNodesResult, error) {
	if len(nodes) == 0 {
		return nil, errors.New("no nodes passed to link nodes")
	}
	parentNode, err := s.client.Object().New(ctx)
	if err != nil {
		return nil, err
	}

	parentPath := path.IpfsPath(parentNode.Cid())
	for _, node := range nodes {
		parentPath, err = s.client.Object().AddLink(ctx, parentPath, node.Name, node.Path)
		if err != nil {
			return nil, errors.Wrap(err, "failed to link nodes")
		}
	}

	return &LinkNodesResult{
		ParentPath: parentPath,
	}, nil
}

func (s *SpaceIpfsClient) PullItem(ctx context.Context, cid cid.Cid) (io.ReadCloser, error) {
	node, err := s.client.Unixfs().Get(ctx, path.IpfsPath(cid))
	if err != nil {
		return nil, err
	}

	var file files.File
	switch f := node.(type) {
	case files.File:
		file = f
	case files.Directory:
		return nil, errors.New("unsupported cid provided")
	default:
		return nil, errors.New("unsupported cid provided")
	}

	return file, nil
}
