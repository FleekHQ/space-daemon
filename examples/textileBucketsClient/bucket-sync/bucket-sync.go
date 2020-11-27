package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"log"
	"os"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	tc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	bc "github.com/textileio/textile/v2/api/bucketsd/client"
	buckets_pb "github.com/textileio/textile/v2/api/bucketsd/pb"
	"github.com/textileio/textile/v2/api/common"
	tb "github.com/textileio/textile/v2/buckets"
	"github.com/textileio/textile/v2/cmd"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type TextileBucketRoot buckets_pb.Root

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

	user2, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		log.Println("error creating user2")
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

	filepath := "file1"
	f := &bytes.Buffer{}
	f.Write([]byte("hello space"))
	_, _, err = buckets.PushPath(ctx, buck.Root.Key, filepath, f)

	if err != nil {
		log.Println("error pushing path")
		log.Fatal(err)
	}

	roles := make(map[string]tb.Role)
	tpk := thread.NewLibp2pPubKey(user2.GetPublic())
	roles[tpk.String()] = tb.Admin
	err = buckets.PushPathAccessRoles(ctx, buck.Root.Key, filepath, roles)
	if err != nil {
		log.Println("error sharing path")
		log.Fatal(err)
	}

	// user 2 tries to access
	ctx1 := context.Background()
	ctx1 = common.NewAPIKeyContext(ctx1, key)
	ctx1, err = common.CreateAPISigContext(ctx1, time.Now().Add(time.Minute*2), secret)
	tok, err = threads.GetToken(ctx1, thread.NewLibp2pIdentity(user2))
	ctx1 = thread.NewTokenContext(ctx1, tok)

	if err != nil {
		log.Println("error creating context")
		log.Fatal(err)
	}

	ctx1 = common.NewThreadNameContext(ctx1, bucket1name)
	ctx1 = common.NewThreadIDContext(ctx1, dbID)
	var buf bytes.Buffer
	err = buckets.PullPath(ctx1, buck.Root.Key, filepath, &buf)
	if err != nil {
		log.Println("error pulling path")
		log.Fatal(err)
	}

	s := buf.String()
	log.Println("fetch file content: " + s)
}
