package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	tc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	bc "github.com/textileio/textile/api/buckets/client"
	buckets_pb "github.com/textileio/textile/api/buckets/pb"
	"github.com/textileio/textile/api/common"
	"github.com/textileio/textile/cmd"
	"google.golang.org/grpc"
)

type TextileBucketRoot buckets_pb.Root

func main() {
	seed := os.Getenv("KEY_SEED")
	threadID := os.Getenv("THREAD_ID")
	host := os.Getenv("TXL_HUB_HOST")
	key := os.Getenv("TXL_USER_KEY")
	secret := os.Getenv("TXL_USER_SECRET")

	var threads *tc.Client
	var buckets *bc.Client
	var err error
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

	ctx := context.Background()
	ctx = common.NewAPIKeyContext(ctx, key)
	ctx, err = common.CreateAPISigContext(ctx, time.Now().Add(time.Minute*2), secret)

	if err != nil {
		log.Println("error creating APISigContext")
		log.Fatal(err)
	}

	sb, err := hex.DecodeString(seed)
	pvk := ed25519.NewKeyFromSeed(sb)
	pbk := make([]byte, 32)
	copy(pbk, pvk[32:])

	var unmarshalledPriv crypto.PrivKey
	var unmarshalledPub crypto.PubKey

	if unmarshalledPriv, err = crypto.UnmarshalEd25519PrivateKey(pvk); err != nil {
		log.Fatal("Cant get libp2p version of priv key")
		return
	}

	if unmarshalledPub, err = crypto.UnmarshalEd25519PublicKey(pbk); err != nil {
		log.Fatal("Cant get libp2p version of pub key")
		return
	}
	log.Println("got libp2p keys")

	tok, err := threads.GetToken(ctx, thread.NewLibp2pIdentity(unmarshalledPriv))
	ctx = thread.NewTokenContext(ctx, tok)

	var pubKeyInBytes []byte
	if pubKeyInBytes, err = unmarshalledPub.Bytes(); err != nil {
		log.Fatal("Cant get bytes of pubkey")
		return
	}

	ctx = common.NewThreadNameContext(ctx, hex.EncodeToString(pubKeyInBytes)+"-personal")

	dbBytes, err := hex.DecodeString(threadID)
	dbID, err := thread.Cast(dbBytes)
	ctx = common.NewThreadIDContext(ctx, dbID)

	log.Println("got thread id ctx")

	bucketList, err := buckets.List(ctx)
	if err != nil {
		log.Fatal("Cant get list of buckets", err)
		return
	}

	result := make([]*TextileBucketRoot, 0)
	for _, r := range bucketList.Roots {
		log.Println("looping through bucket: ", (*TextileBucketRoot)(r).Name)
		if (*TextileBucketRoot)(r).Name == "personal" {
			_, _, err = buckets.PushPath(ctx, (*TextileBucketRoot)(r).Key, fmt.Sprint(int32(time.Now().Unix()))+"synctestfile.md", &bytes.Buffer{})
			result = append(result, (*TextileBucketRoot)(r))
		}
	}
}
