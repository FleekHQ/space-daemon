package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	tc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	bc "github.com/textileio/textile/v2/api/buckets/client"
	buckets_pb "github.com/textileio/textile/v2/api/buckets/pb"
	"github.com/textileio/textile/v2/api/common"
	"github.com/textileio/textile/v2/cmd"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type TextileBucketRoot buckets_pb.Root

type ZeroReader struct{}

func (zr ZeroReader) Read(b []byte) (int, error) {
	b[0] = '0'
	return 1, nil
}

func main() {
	host := os.Getenv("TXL_HUB_TARGET")
	key := os.Getenv("TXL_USER_KEY")
	secret := os.Getenv("TXL_USER_SECRET")

	var threads *tc.Client
	var buckets *bc.Client
	var err error
	auth := common.Credentials{}
	var opts []grpc.DialOption
	hubTarget := host
	threadstarget := host

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

	user1, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		log.Println("error creating user1")
		log.Fatal(err)
	}

	// user 1 creates bucket and adds file

	ctx := context.Background()
	ctx = common.NewAPIKeyContext(ctx, key)

	ctx, err = common.CreateAPISigContext(ctx, time.Now().Add(time.Minute*2), secret)
	if err != nil {
		log.Println("error creating APISigContext")
		log.Fatal(err)
	}

	tok, err := threads.GetToken(ctx, thread.NewLibp2pIdentity(user1))
	if err != nil {
		log.Println("error calling GetToken")
		log.Fatal(err)
	}

	ctx = thread.NewTokenContext(ctx, tok)

	bucket1name := "testbucket1"

	ctx = common.NewThreadNameContext(ctx, bucket1name)
	dbID := thread.NewIDV1(thread.Raw, 32)
	if err := threads.NewDB(ctx, dbID); err != nil {
		log.Println("error calling threads.NewDB")
		log.Fatal(err)
	}

	ctx = common.NewThreadIDContext(ctx, dbID)

	buck, err := buckets.Create(ctx, bc.WithName(bucket1name), bc.WithPrivate(true))
	log.Println("created bucket: " + buck.Root.Name)

	go func() {
		filepath := "infinite"
		zr := &ZeroReader{}

		log.Println("pushing path with zero reader")

		_, _, err = buckets.PushPath(ctx, buck.Root.Key, filepath, zr)
		if err != nil {
			log.Println("error pushing path")
			log.Fatal(err)
		}

		log.Println("pushed path with zero reader")
	}()

	time.Sleep(5 * time.Second)

	log.Println("listing root...")

	result, err := buckets.ListPath(ctx, buck.Root.Key, "")
	if err != nil {
		log.Println("error listing root")
		log.Fatal(err)
		return
	}

	log.Println(fmt.Sprintf("listed root: %+v", result))
}
