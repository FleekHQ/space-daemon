package main

import (
	"bytes"
	"context"
	"log"
	"sort"
	"time"

	"github.com/textileio/go-threads/db"
	"github.com/textileio/go-threads/util"

	"github.com/textileio/go-threads/core/thread"

	"github.com/textileio/textile/api/common"
	"google.golang.org/grpc"

	"github.com/textileio/textile/cmd"

	connmgr "github.com/libp2p/go-libp2p-connmgr"
	tc "github.com/textileio/go-threads/api/client"
	tcore "github.com/textileio/go-threads/core/db"
	bc "github.com/textileio/textile/api/buckets/client"
	"github.com/textileio/textile/core"
)

type BucketSchema struct {
	ID   tcore.InstanceID `json:"_id"`
	Slug string           `json:"slug"`
	DbID string
}

func StartTextile(ctx context.Context) (*core.Textile, error) {
	textile, err := core.NewTextile(ctx, core.Config{
		RepoPath:           ".buckd/repo",
		AddrAPI:            cmd.AddrFromStr("/ip4/127.0.0.1/tcp/3006"),
		AddrAPIProxy:       cmd.AddrFromStr("/ip4/127.0.0.1/tcp/3007"),
		AddrThreadsHost:    cmd.AddrFromStr("/ip4/0.0.0.0/tcp/4006"),
		AddrIPFSAPI:        cmd.AddrFromStr("/ip4/127.0.0.1/tcp/5001"),
		AddrGatewayHost:    cmd.AddrFromStr("/ip4/127.0.0.1/tcp/8006"),
		AddrGatewayURL:     "http://127.0.0.1:8006",
		AddrMongoURI:       "mongodb://127.0.0.1:27027",
		MongoName:          "buckets",
		ThreadsConnManager: connmgr.NewConnManager(10, 50, time.Second*20),
		Debug:              false,
	})
	if err != nil {
		return nil, err
	}

	textile.Bootstrap()

	return textile, nil
}

func main() {
	bucketName := "DefaultBucket"
	host := "127.0.0.1:3006"
	ctx, _ := context.WithCancel(context.Background())
	_, err := StartTextile(ctx)
	if err != nil {
		log.Fatalf("Failed to start textile: %+v", err)
	}

	threadsClient, err := tc.NewClient(host,
		grpc.WithInsecure(),
		grpc.WithPerRPCCredentials(common.Credentials{}),
	)
	if err != nil {
		log.Fatalf("Failed to create threads client: %+v", err)
	}

	dbId := thread.NewIDV1(thread.Raw, 32)
	ctx = common.NewThreadIDContext(ctx, dbId)
	if err := threadsClient.NewDB(ctx, dbId); err != nil {
		log.Fatalf("Failed creating bucket[%s]: %+v", bucketName, err)
	}

	if err := threadsClient.NewCollection(ctx, dbId, db.CollectionConfig{
		Name:    "BucketsMetadata",
		Schema:  util.SchemaFromInstance(&BucketSchema{}, false),
		Indexes: nil,
	}); err != nil {
		log.Fatalf("Failed creating bucket collection: %+v", err)
	}

	bucketClient, err := bc.NewClient(
		host,
		grpc.WithInsecure(),
		grpc.WithPerRPCCredentials(common.Credentials{}),
	)
	if err != nil {
		log.Fatalf("Failed to create bucket client: %+v", err)
	}

	bucket, err := bucketClient.Init(ctx, bc.WithName(bucketName), bc.WithPrivate(true))
	if err != nil {
		log.Fatalf("Failed to created default bucket: %+v", err)
	}

	fileContent := &bytes.Buffer{}
	fileContent.WriteString("Random text content")
	_, _, err = bucketClient.PushPath(ctx, bucket.Root.Key, "parentFolder/a.txt", fileContent)
	if err != nil {
		log.Fatalf("Failed uploading file to parentFolder/a.txt: %+v", err)
	}

	listReply, err := bucketClient.ListPath(ctx, bucket.Root.Key, "")
	if err != nil {
		log.Fatalf("Listing Paths failed: %+v", err)
	}

	parentFolderPos := sort.Search(len(listReply.Item.Items), func(i int) bool {
		return listReply.Item.Items[i].Name == "parentFolder"
	})
	if parentFolderPos == len(listReply.Item.Items) {
		log.Fatalf("Error: parentFolder not found")
	}

	parentFolderItem := listReply.Item.Items[parentFolderPos]
	if parentFolderItem.IsDir == false {
		log.Fatalf("parentFolder's ListPathItem.IsDir should be 'true', but got false")
	}

	log.Printf("ParentFolder Name: %s\nIsDir: %v\n", parentFolderItem.Name, parentFolderItem.IsDir)
}
