package main

import (
	"context"
	"fmt"
	"os"

	ma "github.com/multiformats/go-multiaddr"
	tc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/textile/api/common"
	"github.com/textileio/textile/cmd"
	"google.golang.org/grpc"
)

func main() {
	addr := os.Args[2]
	key := os.Args[3]

	m1, _ := ma.NewMultiaddr(addr)

	var threads *tc.Client
	host := "127.0.0.1:3006"
	auth := common.Credentials{}
	var opts []grpc.DialOption
	threadstarget := host
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithPerRPCCredentials(auth))
	threads, err := tc.NewClient(threadstarget, opts...)
	if err != nil {
		cmd.Fatal(err)
	}

	threadCtx := context.Background()
	k, err := thread.KeyFromString(key)

	sk := k.Service()
	rk := k.Read()

	threads.NewDBFromAddr(threadCtx, m1, rk)

	db, err := threads.ListDBs(threadCtx)

	for k, v := range db {
		fmt.Println("looping through thread id: ", k)
		fmt.Println("db info: ", v)
	}
}
