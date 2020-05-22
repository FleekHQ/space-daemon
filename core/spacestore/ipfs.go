package spacestore

import (
	"context"
	ipfslite "github.com/hsanjuan/ipfs-lite"
	datastore "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multiaddr"
)

func createIpfsPeer(ctx context.Context) (*ipfslite.Peer, error) {
	host, err := libp2p.New(ctx)
	if err != nil {
		return nil, err
	}

	// TODO: Pipe in host public and private key
	hostKey, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		return nil, err
	}

	// listen on all interface
	listen, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")

	host, dht, err := ipfslite.SetupLibp2p(
		ctx,
		hostKey,
		nil,
		[]multiaddr.Multiaddr{listen},
		nil, // using in-memory datastore for now
	)

	if err != nil {
		return nil, err
	}

	datastore := dssync.MutexWrap(datastore.NewMapDatastore())
	ipfsPeer, err := ipfslite.New(ctx, datastore, host, dht, nil)
	if err != nil {
		return nil, err
	}

	ipfsPeer.Bootstrap(ipfslite.DefaultBootstrapPeers())
	return ipfsPeer, nil
}
