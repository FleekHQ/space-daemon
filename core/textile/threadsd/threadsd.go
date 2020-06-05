package textilethreadsd

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os/user"
	"path/filepath"
	"time"

	"github.com/FleekHQ/space-poc/log"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/textileio/go-threads/api"
	tpb "github.com/textileio/go-threads/api/pb"
	tCommon "github.com/textileio/go-threads/common"
	netapi "github.com/textileio/go-threads/net/api"
	netpb "github.com/textileio/go-threads/net/api/pb"
	"github.com/textileio/go-threads/util"
	"google.golang.org/grpc"
)

const (
	p2pMultiAddr       = "/ip4/0.0.0.0/tcp/4006"
	hostMultiAddr      = "/ip4/127.0.0.1/tcp/6006"
	hostProxyMultiAddr = "/ip4/127.0.0.1/tcp/6007"
)

type TextileThreadsd struct {
	isRunning bool
	Ready     chan bool
	proxy     *http.Server
	server    *grpc.Server
	n         tCommon.NetBoostrapper
}

func (tt *TextileThreadsd) Start() {
	hostAddr, err := ma.NewMultiaddr(p2pMultiAddr)
	if err != nil {
		log.Fatal(err)
	}
	apiAddr, err := ma.NewMultiaddr(hostMultiAddr)
	if err != nil {
		log.Fatal(err)
	}
	apiProxyAddr, err := ma.NewMultiaddr(hostProxyMultiAddr)
	if err != nil {
		log.Fatal(err)
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	repo := filepath.Join(usr.HomeDir, ".threads")
	debug := false

	tt.n, err = tCommon.DefaultNetwork(
		repo,
		tCommon.WithNetHostAddr(hostAddr),
		tCommon.WithConnectionManager(connmgr.NewConnManager(100, 400, time.Second*20)),
		tCommon.WithNetDebug(debug))
	if err != nil {
		log.Fatal(err)
	}
	tt.n.Bootstrap(util.DefaultBoostrapPeers())

	service, err := api.NewService(tt.n, api.Config{
		RepoPath: repo,
		Debug:    debug,
	})
	if err != nil {
		log.Fatal(err)
	}
	netService, err := netapi.NewService(tt.n, netapi.Config{
		Debug: debug,
	})
	if err != nil {
		log.Fatal(err)
	}

	target, err := util.TCPAddrFromMultiAddr(apiAddr)
	if err != nil {
		log.Fatal(err)
	}
	ptarget, err := util.TCPAddrFromMultiAddr(apiProxyAddr)
	if err != nil {
		log.Fatal(err)
	}

	tt.server = grpc.NewServer()
	listener, err := net.Listen("tcp", target)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		tpb.RegisterAPIServer(tt.server, service)
		netpb.RegisterAPIServer(tt.server, netService)
		if err := tt.server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Error("Threadsd grpc server error: ", err)
		}
	}()
	webrpc := grpcweb.WrapServer(
		tt.server,
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
		grpcweb.WithWebsockets(true),
		grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool {
			return true
		}))
	tt.proxy = &http.Server{
		Addr: ptarget,
	}
	tt.proxy.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if webrpc.IsGrpcWebRequest(r) ||
			webrpc.IsAcceptableGrpcCorsRequest(r) ||
			webrpc.IsGrpcWebSocketRequest(r) {
			webrpc.ServeHTTP(w, r)
		}
	})
	go func() {
		if err := tt.proxy.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Threasdd grpc proxy error: ", err)
		}
	}()

	log.Info("threadsd: Your peer ID is " + tt.n.Host().ID().String())

	tt.isRunning = true
	tt.Ready <- true
}

func (tt *TextileThreadsd) WaitForReady() chan bool {
	return tt.Ready
}
func (tt *TextileThreadsd) Stop() {
	tt.isRunning = false
	close(tt.Ready)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := tt.proxy.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	tt.server.GracefulStop()
	if err := tt.n.Close(); err != nil {
		log.Fatal(err)
	}

	tt.proxy = nil
	tt.server = nil
	tt.n = nil
}
