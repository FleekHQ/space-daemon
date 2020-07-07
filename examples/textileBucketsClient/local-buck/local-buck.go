package main

import (
	"context"
	"fmt"
	"os"
	"os/user"

	"github.com/FleekHQ/space-daemon/log"
	tc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	nc "github.com/textileio/go-threads/net/api/client"
	bc "github.com/textileio/textile/api/buckets/client"
	"github.com/textileio/textile/api/common"
	"github.com/textileio/textile/cmd"
	"github.com/textileio/textile/core"
	"google.golang.org/grpc"
)

var IpfsAddr string
var MongoUsr string
var MongoPw string
var MongoHost string

func main() {

	IpfsAddr = os.Getenv("IPFS_ADDR")
	MongoUsr = os.Getenv("MONGO_USR")
	MongoPw = os.Getenv("MONGO_PW")
	MongoHost = os.Getenv("MONGO_HOST")

	addrAPI := cmd.AddrFromStr("/ip4/127.0.0.1/tcp/3006")
	addrAPIProxy := cmd.AddrFromStr("/ip4/127.0.0.1/tcp/3007")
	addrThreadsHost := cmd.AddrFromStr("/ip4/0.0.0.0/tcp/4006")
	addrIpfsAPI := cmd.AddrFromStr(IpfsAddr)

	addrGatewayHost := cmd.AddrFromStr("/ip4/127.0.0.1/tcp/8006")
	addrGatewayURL := "http://127.0.0.1:8006"
	fmt.Println("mongo host: ", MongoHost)
	addrMongoURI := "mongodb+srv://" + MongoUsr + ":" + MongoPw + "@" + MongoHost

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	textile, err := core.NewTextile(ctx, core.Config{
		RepoPath:        usr.HomeDir + "/.buckd/repo",
		AddrAPI:         addrAPI,
		AddrAPIProxy:    addrAPIProxy,
		AddrThreadsHost: addrThreadsHost,
		AddrIPFSAPI:     addrIpfsAPI,
		AddrGatewayHost: addrGatewayHost,
		AddrGatewayURL:  addrGatewayURL,
		AddrMongoURI:    addrMongoURI,
		MongoName:       "buckets",
		Debug:           false,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer textile.Close()
	textile.Bootstrap()

	fmt.Println("Welcome to Buckets!")
	fmt.Println("Your peer ID is " + textile.HostID().String())

	// now create a bucket on that thread
	var threads *tc.Client
	var buckets *bc.Client
	var netc *nc.Client
	host := "127.0.0.1:3006"
	auth := common.Credentials{}
	var opts []grpc.DialOption
	hubTarget := host
	threadstarget := host
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithPerRPCCredentials(auth))

	buckets, err = bc.NewClient(hubTarget, opts...)
	if err != nil {
		cmd.Fatal(err)
	}
	threads, err = tc.NewClient(threadstarget, opts...)
	if err != nil {
		cmd.Fatal(err)
	}
	netc, err = nc.NewClient(host, opts...)

	log.Info("Finished client init, calling user init ...")

	threadCtx := context.Background()
	threadCtx = common.NewThreadNameContext(threadCtx, "testthreadname")
	dbID := thread.NewIDV1(thread.Raw, 32)
	if err := threads.NewDB(threadCtx, dbID); err != nil {
		log.Info("error calling threads.NewDB")
		log.Fatal(err)
	}

	ctx = common.NewThreadIDContext(threadCtx, dbID)

	buck, err := buckets.Init(ctx, bc.WithName("personal"), bc.WithPrivate(true))
	fmt.Println("info: ", buck)

	db, err := threads.ListDBs(ctx)

	fmt.Println("got back from listdbs")

	for k, v := range db {
		fmt.Println("looping through thread id: ", k)
		fmt.Println("db info: ", v)

		// replicate on hub
		netc.AddReplicator(ctx, dbID, cmd.AddrFromStr("/ip4/54.188.82.109/tcp/3006"))
	}
}
