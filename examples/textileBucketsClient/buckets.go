package main

import (
	"context"
	"log"
	"time"

	tc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	bc "github.com/textileio/textile/api/buckets/client"
	"github.com/textileio/textile/api/common"
)

const ctxTimeout = 30000

func authCtx(duration time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	return ctx, cancel
}

func threadCtx(duration time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := authCtx(ctxTimeout)
	ctx = common.NewThreadIDContext(ctx, getThreadID())
	return ctx, cancel
}

func createBucket(user string, bucketSlug string) {
	// create thread
	var threads *tc.Client // should come from rest context
	var buckets *bc.Client // should come from rest context
	var dbID thread.ID
	dbID = thread.NewIDV1(thread.Raw, 32)

	ctx, cancel := threadCtx(ctxTimeout)
	defer cancel()
	ctx = common.NewThreadNameContext(ctx, user+"-"+bucketSlug)
	if err := threads.NewDB(ctx, dbID); err != nil {
		log.Fatal(err)
	}

	// use thread to create bucket
	buck, err := buckets.Init(ctx, bucketSlug)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.Println("hello world textile!")
}
