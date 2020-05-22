package main

import (
	"context"
	"crypto/rand"
	"log"
	"os"
	"time"

	crypto "github.com/libp2p/go-libp2p-crypto"
	tc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	bc "github.com/textileio/textile/api/buckets/client"
	pb "github.com/textileio/textile/api/buckets/pb"
	"github.com/textileio/textile/api/common"
	"github.com/textileio/textile/cmd"
	"google.golang.org/grpc"
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

func initUser(threads *tc.Client, buckets *bc.Client, user string, bucketSlug string) *pb.InitReply {
	// TODO: this should be happening in an auth lambda
	// only needed for hub connections
	key := os.Getenv("TXL_USER_KEY")
	secret := os.Getenv("TXL_USER_SECRET")
	ctx := context.Background()
	ctx = common.NewAPIKeyContext(ctx, key)
	ctx, err := common.CreateAPISigContext(ctx, time.Now().Add(time.Minute), secret)

	if err != nil {
		log.Println("error creating APISigContext")
		log.Fatal(err)
	}

	// TODO: get from key manager instead
	sk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	// TODO: CTX has to be made from session key received from lambda
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
	buck, err := buckets.Init(ctx, bucketSlug)

	return buck
}

func main() {
	log.Println("hello world textile!")

	var threads *tc.Client
	var buckets *bc.Client
	// might need these for other ops so leaving here as commented
	// out and below
	// var users *uc.Client
	// var hub *hc.Client
	var err error

	auth := common.Credentials{}
	var opts []grpc.DialOption
	hubTarget := "127.0.0.1:3006"
	threadstarget := "127.0.0.1:3006"
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
	// hub, err = hc.NewClient(hubTarget, opts...)
	// if err != nil {
	// 	cmd.Fatal(err)
	// }
	// users, err = uc.NewClient(hubTarget, opts...)
	// if err != nil {
	// 	cmd.Fatal(err)
	// }

	log.Println("Finished client init, calling user init ...")

	res := initUser(threads, buckets, "test-user", "test-bucket")
	log.Println(res)
}
