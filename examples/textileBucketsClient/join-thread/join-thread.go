package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/user"

	ma "github.com/multiformats/go-multiaddr"
	tc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/textile/v2/api/common"
	"github.com/textileio/textile/v2/cmd"
	"github.com/textileio/textile/v2/core"
	"google.golang.org/grpc"
)

var IpfsAddr string
var MongoUsr string
var MongoPw string
var MongoHost string
var MongoRepSet string

func main() {
	IpfsAddr = os.Getenv("IPFS_ADDR")
	MongoUsr = os.Getenv("MONGO_USR")
	MongoPw = os.Getenv("MONGO_PW")
	MongoHost = os.Getenv("MONGO_HOST")
	MongoRepSet = os.Getenv("MONGO_REPLICA_SET")

	addrAPI := cmd.AddrFromStr("/ip4/127.0.0.1/tcp/3006")
	addrAPIProxy := cmd.AddrFromStr("/ip4/127.0.0.1/tcp/3007")
	addrThreadsHost := cmd.AddrFromStr("/ip4/0.0.0.0/tcp/4006")

	addrIpfsAPI := cmd.AddrFromStr(IpfsAddr)

	addrGatewayHost := cmd.AddrFromStr("/ip4/127.0.0.1/tcp/8006")
	addrGatewayURL := "http://127.0.0.1:8006"

	fmt.Println("mongo host: ", MongoHost)
	addrMongoURI := "mongodb://" + MongoUsr + ":" + MongoPw + "@" + MongoHost + "/?ssl=true&replicaSet=" + MongoRepSet + "&authSource=admin&retryWrites=true&w=majority"

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
	defer textile.Close(false)
	textile.Bootstrap()

	fmt.Println("Welcome to Buckets!")
	fmt.Println("Your peer ID is " + textile.HostID().String())

	fmt.Println("starting join thread")

	addr := os.Getenv("JOIN_THREAD_ADDR")
	key := os.Getenv("JOIN_THREAD_KEY")

	m1, _ := ma.NewMultiaddr(addr)

	var threads *tc.Client
	host := "127.0.0.1:3006"
	auth := common.Credentials{}
	var opts []grpc.DialOption
	threadstarget := host
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithPerRPCCredentials(auth))
	threads, err = tc.NewClient(threadstarget, opts...)
	if err != nil {
		cmd.Fatal(err)
	}

	threadCtx := context.Background()
	k, err := thread.KeyFromString(key)

	err = threads.NewDBFromAddr(threadCtx, m1, k)

	if err != nil {
		fmt.Println("error new db from addr: ", err)
	}

	db, err := threads.ListDBs(threadCtx)

	fmt.Println("about to loop thru dbs: ", db)

	for k, v := range db {
		fmt.Println("looping through thread id: ", k)
		fmt.Println("db info: ", v)
	}
}
