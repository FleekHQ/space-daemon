package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	crypto "github.com/libp2p/go-libp2p-crypto"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/textileio/go-threads/api"
	tc "github.com/textileio/go-threads/api/client"
	tpb "github.com/textileio/go-threads/api/pb"
	tCommon "github.com/textileio/go-threads/common"
	"github.com/textileio/go-threads/core/thread"
	netapi "github.com/textileio/go-threads/net/api"
	netapiclient "github.com/textileio/go-threads/net/api/client"
	netpb "github.com/textileio/go-threads/net/api/pb"
	"github.com/textileio/go-threads/util"
	bc "github.com/textileio/textile/api/buckets/client"
	pb "github.com/textileio/textile/api/buckets/pb"
	"github.com/textileio/textile/api/common"
	uc "github.com/textileio/textile/api/users/client"
	"github.com/textileio/textile/cmd"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const ctxTimeout = 30

func authCtx(duration time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	return ctx, cancel
}

// these next 2 helpers are from the lib but wasnt
// sure how to export them
func threadCtx(duration time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := authCtx(duration)
	ctx = common.NewThreadIDContext(ctx, getThreadID())
	return ctx, cancel
}

func getThreadID() (id thread.ID) {
	// get from Space config instead
	idstr := os.Getenv("thread")
	if idstr != "" {
		var err error
		id, err = thread.Decode(idstr)
		if err != nil {
			cmd.Fatal(err)
		}
	}
	return
}

func runThreadsLocally() {
	hostAddr, err := ma.NewMultiaddr("/ip4/0.0.0.0/tcp/4006")
	if err != nil {
		log.Fatal(err)
	}
	apiAddr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/6006")
	if err != nil {
		log.Fatal(err)
	}
	apiProxyAddr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/6007")
	if err != nil {
		log.Fatal(err)
	}

	repo := ".threads"
	debug := false

	n, err := tCommon.DefaultNetwork(
		repo,
		tCommon.WithNetHostAddr(hostAddr),
		tCommon.WithConnectionManager(connmgr.NewConnManager(100, 400, time.Second*20)),
		tCommon.WithNetDebug(debug))
	if err != nil {
		log.Fatal(err)
	}
	defer n.Close()
	n.Bootstrap(util.DefaultBoostrapPeers())
	service, err := api.NewService(n, api.Config{
		RepoPath: repo,
		Debug:    debug,
	})
	if err != nil {
		log.Fatal(err)
	}
	netService, err := netapi.NewService(n, netapi.Config{
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

	server := grpc.NewServer()
	listener, err := net.Listen("tcp", target)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		tpb.RegisterAPIServer(server, service)
		netpb.RegisterAPIServer(server, netService)
		if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Fatalf("serve error: %v", err)
		}
	}()
	webrpc := grpcweb.WrapServer(
		server,
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
		grpcweb.WithWebsockets(true),
		grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool {
			return true
		}))
	proxy := &http.Server{
		Addr: ptarget,
	}
	proxy.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if webrpc.IsGrpcWebRequest(r) ||
			webrpc.IsAcceptableGrpcCorsRequest(r) ||
			webrpc.IsGrpcWebSocketRequest(r) {
			webrpc.ServeHTTP(w, r)
		}
	})
	go func() {
		if err := proxy.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("proxy error: %v", err)
		}
	}()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := proxy.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
		server.GracefulStop()
		if err := n.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	fmt.Println("Welcome to Threads!")
	fmt.Println("Your peer ID is " + n.Host().ID().String())

	log.Println("threadsd started")

	select {}
}

type Bucket struct {
	Key       string `json:"_id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	DNSRecord string `json:"dns_record,omitempty"`
	//Archives  Archives `json:"archives"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

func initUser(threads *tc.Client, buckets *bc.Client, users *uc.Client, netclient *netapiclient.Client, user string, bucketSlug string) *pb.InitReply {
	// only needed for hub connections

	key := os.Getenv("TXL_USER_KEY")
	secret := os.Getenv("TXL_USER_SECRET")

	// TODO: this should be happening in an auth lambda
	ctx := context.Background()
	ctx = common.NewAPIKeyContext(ctx, key)
	ctx, err := common.CreateAPISigContext(ctx, time.Now().Add(time.Minute*2), secret)

	if err != nil {
		log.Println("error creating APISigContext")
		log.Fatal(err)
	}

	// TODO: get from key manager instead
	sk, _, err := crypto.GenerateEd25519Key(rand.Reader)

	// TODO: CTX has to be made from session key received from lambda
	// ctx on next line needs to be rebuilt from the authorization from the lambda
	tok, err := threads.GetToken(ctx, thread.NewLibp2pIdentity(sk))
	ctx = thread.NewTokenContext(ctx, tok)

	// create thread
	ctx = common.NewThreadNameContext(ctx, user+"-"+bucketSlug)
	dbID := thread.NewIDV1(thread.Raw, 32)
	// TODO: store threadid in config
	if err := threads.NewDB(ctx, dbID); err != nil {
		log.Println("error calling threads.NewDB")
		log.Fatal(err)
	}

	ctx = common.NewThreadIDContext(ctx, dbID)
	// create bucket
	buck, err := buckets.Init(ctx, bc.WithName(bucketSlug), bc.WithPrivate(true))
	buckets.Init(ctx, bc.WithName(bucketSlug+"2"), bc.WithPrivate(true))

	log.Println("finished creating bucket")

	hostid, err := netclient.GetHostID(ctx)
	if err != nil {
		log.Println("error getting HOST ID: ", err)
	}
	log.Println("HOSTID: ", hostid)

	newCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	opt := tc.ListenOption{}

	mid, err := users.SetupMailbox(newCtx)
	if err != nil {
		log.Println("Unable to setup mailbox", err)
		return nil
	}
	log.Println("Mailbox id: ", mid.String())

	//listPath on a folder that doesnt exist
	lp, err := buckets.ListPath(ctx, buck.Root.Key, "random/folderA/doesntexists")
	if err != nil {
		log.Println("error doing list path on non existent directoy: ", err)
	}
	log.Println("lp1: ", lp)

	db, err := users.ListThreads(ctx)

	if err != nil {
		fmt.Println("error getting list dbs")
		fmt.Println(err)
	}

	fmt.Println("listing dbs")
	for k, v := range db.GetList() {
		fmt.Println("looping through thread id: ", k)
		fmt.Println("db info: ", v)
	}

	emptyDirPath := strings.TrimRight("dummy", "/") + "/" + ".keep"
	_, _, err = buckets.PushPath(ctx, buck.Root.Key, emptyDirPath, &bytes.Buffer{})

	//listPath on a folder that exists
	r := strings.NewReader("IPFS test data for reader")
	r2 := strings.NewReader("IPFS test data  ./tfor reader2")
	buckets.PushPath(ctx, buck.Root.Key, "another/folderB/file1", r)
	buckets.PushPath(ctx, buck.Root.Key, "another/folderB/file2", r2)
	lp, err = buckets.ListPath(ctx, buck.Root.Key, "another/folderB")
	if err != nil {
		log.Println("error doing list path on non existent directoy: ", err)
	}
	log.Println("lp2: ", lp)

	// put in go routine
	channel, err := threads.Listen(newCtx, dbID, []tc.ListenOption{opt})

	log.Println("finished creating channel")

	if err != nil {
		log.Fatalf("failed to call listen: %v", err)
	}

	go func() {
		time.Sleep(time.Second)
		buckets.Init(ctx, bc.WithName(bucketSlug+"3"), bc.WithPrivate(true))
	}()

	// a separete go routine that keeps checking if msgs are there
	// and calls handler function
	val, ok := <-channel

	if !ok {
		log.Println("channel no longer active at first events")
	} else {
		log.Println("received from channel!!!!")
		log.Println(val)
		instance := &Bucket{}
		if err = json.Unmarshal(val.Action.Instance, instance); err != nil {
			log.Fatalf("failed to unmarshal listen result: %v", err)
		}

		log.Printf("instance: %+v", *instance)
	}

	val, ok = <-channel

	if !ok {
		log.Println("channel 2 no longer active at first events")
	} else {
		log.Println("received 2 from channel!!!!")
		log.Println(val)
	}

	log.Println("finished creating channel")

	if err != nil {
		log.Fatalf("failed to call listen: %v", err)
	}

	val, ok = <-channel

	if !ok {
		log.Println("channel no longer active at first events")
	} else {
		log.Println("received from channel!!!!")
		log.Println(val)
	}

	val, ok = <-channel

	if !ok {
		log.Println("channel 2 no longer active at first events")
	} else {
		log.Println("received 2 from channel!!!!")
		log.Println(val)
	}

	return buck
}

func main() {
	mode := os.Args[1]

	if mode == "threads" {
		log.Println("running in process threads")
		runThreadsLocally()
		return
	}

	if mode == "hub" {
		var threads *tc.Client
		var buckets *bc.Client
		// might need these for other ops so leaving here as commented
		// out and below
		var users *uc.Client
		// var hub *hc.Client
		var err error

		host := os.Getenv("TXL_HUB_TARGET")
		threadstarget := os.Getenv("TXL_THREADS_TARGET")
		fmt.Println("hub host: " + host)
		fmt.Println("threads host: " + threadstarget)

		auth := common.Credentials{}
		var opts []grpc.DialOption
		hubTarget := host

		if strings.Contains(host, "443") {
			creds := credentials.NewTLS(&tls.Config{})
			opts = append(opts, grpc.WithTransportCredentials(creds))
			auth.Secure = true
		} else {
			opts = append(opts, grpc.WithInsecure())
		}
		opts = append(opts, grpc.WithPerRPCCredentials(auth))

		buckets, err = bc.NewClient(hubTarget, opts...)
		if err != nil {
			cmd.Fatal(err)
		}
		threads, err = tc.NewClient(threadstarget, opts...)
		if err != nil {
			cmd.Fatal(err)
		}

		users, err = uc.NewClient(hubTarget, opts...)
		if err != nil {
			cmd.Fatal(err)
		}

		netclient, err := netapiclient.NewClient(host, opts...)
		if err != nil {
			cmd.Fatal(err)
		}

		log.Println("Finished client init, calling user init ...")

		// hub
		res := initUser(threads, buckets, users, netclient, "test-user", "test-bucket")
		log.Println(res)
	}
}
