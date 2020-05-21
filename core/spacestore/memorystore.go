package spacestore

import (
	"context"
	ipfslite "github.com/hsanjuan/ipfs-lite"
	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	"github.com/pkg/errors"
	"log"
)

// Memory Spacestore is an in-memory implementation fo the SpaceStore
// It uses the ipfs peer to fetch data and caches them in memory
// on restart all in-memory updates are lost
type MemorySpaceStore struct {
	peer *ipfslite.Peer
	folderCid cid.Cid
	storage map[string]string
}

func NewMemoryStore(ctx context.Context) (*MemorySpaceStore, error) {
	ipfspeer, err := createIpfsPeer(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating ipfs-lite peer")
	}

	folderCid, err := cid.Parse("QmZd7HwfHS15PdDeitZ7tdpfTq5CPFuQZysydQsTpUu8Kr")
	if err != nil {
		return nil, err
	}

	return &MemorySpaceStore{
		peer: ipfspeer,
		folderCid: folderCid,
		storage: make(map[string]string),
	}, nil
}

func (m *MemorySpaceStore) Get(ctx context.Context, path string) (format.Node, error) {
	log.Printf("Get path: %s in folder %s", path, m.folderCid.String())
	return m.peer.Get(ctx, m.folderCid)
}