package ipfs

import (
	"context"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/log"

	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	icore "github.com/ipfs/interface-go-ipfs-core"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"

	ipfsconfig "github.com/ipfs/go-ipfs-config"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/go-ipfs/commands"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/corehttp"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/libp2p/go-libp2p-core/peer"
)

type IpfsNode struct {
	coreApi   coreiface.CoreAPI
	coreNode  *core.IpfsNode
	cancel    context.CancelFunc

	IsRunning bool
	Ready     chan bool
	cfg       config.Config
}

func NewIpsNode(cfg config.Config) *IpfsNode {
	return &IpfsNode{
		Ready: make(chan bool),
		cfg:   cfg,
	}
}

func (node *IpfsNode) Start(ctx context.Context) error {
	log.Info("Starting the ipfs node")

	err := node.start()
	if err != nil {
		return err
	}

	log.Info("Running the ipfs node")

	node.IsRunning = true
	node.Ready <- true

	return nil
}

func (node *IpfsNode) WaitForReady() chan bool {
	return node.Ready
}

func (node *IpfsNode) Stop() error {
	node.IsRunning = false

	err := node.stop()
	if err != nil {
		return err
	}

	close(node.Ready)

	return nil
}

func (node *IpfsNode) Shutdown() error {
	return node.Stop()
}

func (node *IpfsNode) start() error {
	ctx, cancel := context.WithCancel(context.Background())
	node.cancel = cancel

	pathRoot, err := ipfsconfig.PathRoot()
	if err != nil {
		return err
	}

	repoPath := node.cfg.GetString(config.Ipfsnodepath, pathRoot)
	if repoPath == "" {
		repoPath = pathRoot
	}

	if err := setupPlugins(repoPath); err != nil {
		return err
	}

	// init the repo
	repoCfg, err := ipfsconfig.Init(ioutil.Discard, 2048)
	if err != nil {
		return err
	}

	err = fsrepo.Init(repoPath, repoCfg)
	if err != nil {
		return err
	}

	// open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return err
	}

	// construct the node
	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTClientOption,
		Repo:    repo,
	}

	node.coreNode, err = core.NewNode(ctx, nodeOptions)
	if err != nil {
		return err
	}

	node.coreApi, err = coreapi.NewCoreAPI(node.coreNode)
	if err != nil {
		return err
	}

	addr := node.cfg.GetString(config.Ipfsnodeaddr, "/ip4/127.0.0.1/tcp/5001")

	var opts = []corehttp.ServeOption{
		corehttp.GatewayOption(true, "/ipfs", "/ipns"),
		corehttp.WebUIOption,
		corehttp.CommandsOption(cmdCtx(node.coreNode, repoPath)),
	}

	go func() {
		if err := corehttp.ListenAndServe(node.coreNode, addr, opts...); err != nil {
			return
		}
	}()

	// TODO: better place?
	bootstrapNodes := []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
		"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		"/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	}

	go connectToPeers(ctx, node.coreApi, bootstrapNodes)

	return nil
}

func (node *IpfsNode) stop() error {
	node.cancel()

	return nil
}

func connectToPeers(ctx context.Context, ipfs icore.CoreAPI, peers []string) error {
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peerstore.PeerInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		pii, err := peerstore.InfoFromP2pAddr(addr)
		if err != nil {
			return err
		}
		pi, ok := peerInfos[pii.ID]
		if !ok {
			pi = &peerstore.PeerInfo{ID: pii.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}

	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peerstore.PeerInfo) {
			defer wg.Done()
			err := ipfs.Swarm().Connect(ctx, *peerInfo)
			if err != nil {
				return
			}
		}(peerInfo)
	}
	wg.Wait()
	return nil
}

func setupPlugins(externalPluginsPath string) error {
	// load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

func cmdCtx(node *core.IpfsNode, repoPath string) commands.Context {
	return commands.Context{
		ConfigRoot: repoPath,
		LoadConfig: func(path string) (*ipfsconfig.Config, error) {
			return node.Repo.Config()
		},
		ConstructNode: func() (*core.IpfsNode, error) {
			return node, nil
		},
	}
}
