package textile

import (
	"context"
	"io"
	"sync"

	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/textileio/go-threads/core/thread"
	bucketsClient "github.com/textileio/textile/api/buckets/client"
	bucketsproto "github.com/textileio/textile/api/buckets/pb"
)

type BucketsClient interface {
	PushPath(ctx context.Context, key, pth string, reader io.Reader, opts ...bucketsClient.Option) (result path.Resolved, root path.Resolved, err error)
	PullPath(ctx context.Context, key, pth string, writer io.Writer, opts ...bucketsClient.Option) error
	ListPath(ctx context.Context, key, pth string) (*bucketsproto.ListPathReply, error)
	RemovePath(ctx context.Context, key, pth string, opts ...bucketsClient.Option) (path.Resolved, error)
}

// NOTE: all write operations should use the lock for the bucket to keep consistency
// TODO: Maybe read operations dont need a lock, needs testing
// struct for implementing bucket interface
type bucket struct {
	lock          sync.RWMutex
	root          *bucketsproto.Root
	client        Client
	bucketsClient BucketsClient
}

func (b *bucket) Slug() string {
	return b.GetData().Name
}

func newBucket(root *bucketsproto.Root, client Client, bucketsClient BucketsClient) *bucket {
	return &bucket{
		root:          root,
		client:        client,
		bucketsClient: bucketsClient,
	}
}

func (b *bucket) Key() string {
	return b.GetData().Key
}

func (b *bucket) GetData() BucketData {
	return BucketData{
		Key:       b.root.Key,
		Name:      b.root.Name,
		Path:      b.root.Path,
		DNSRecord: "",
		CreatedAt: b.root.CreatedAt,
		UpdatedAt: b.root.UpdatedAt,
	}
}

func (b *bucket) GetContext(ctx context.Context) (context.Context, *thread.ID, error) {
	return b.client.GetLocalBucketContext(ctx, b.Slug())
}
