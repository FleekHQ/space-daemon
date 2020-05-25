package fs_data_source

import (
	"context"
	"log"

	"github.com/libp2p/go-libp2p-core/peer"

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
	listen := multiaddr.StringCast("/ip4/0.0.0.0/tcp/0")

	host, dht, err := ipfslite.SetupLibp2p(
		ctx,
		hostKey,
		nil,
		[]multiaddr.Multiaddr{listen},
		nil, // using in-memory ds for now
	)

	if err != nil {
		return nil, err
	}
	log.Printf("Host information:\nID: %s\nAddresses: %+v", host.ID(), host.Addrs())

	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	ipfsPeer, err := ipfslite.New(ctx, ds, host, dht, nil)
	if err != nil {
		return nil, err
	}

	// These are Perfects Local host configuration
	ipfsPeer.Bootstrap([]peer.AddrInfo{
		{
			ID: "QmaYPM3zNiNPxUPFZrzBiz2VCJZSBU83EkySABtbbMrf9Q",
			Addrs: []multiaddr.Multiaddr{
				//multiaddr.StringCast("/ip4/127.0.0.1/tcp/4001/p2p/QmaYPM3zNiNPxUPFZrzBiz2VCJZSBU83EkySABtbbMrf9Q"),
				multiaddr.StringCast("/ip4/192.168.1.70/tcp/4001/ipfs/QmaYPM3zNiNPxUPFZrzBiz2VCJZSBU83EkySABtbbMrf9Q"),
				//multiaddr.StringCast("/ip6/::1/tcp/4001/p2p/QmaYPM3zNiNPxUPFZrzBiz2VCJZSBU83EkySABtbbMrf9Q"),
				//multiaddr.StringCast("/ip4/41.76.196.253/tcp/4001/p2p/QmaYPM3zNiNPxUPFZrzBiz2VCJZSBU83EkySABtbbMrf9Q"),
			},
		},
	})
	//ipfsPeer.Bootstrap(ipfslite.DefaultBootstrapPeers())
	return ipfsPeer, nil
}
