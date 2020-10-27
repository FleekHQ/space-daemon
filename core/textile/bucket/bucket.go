package bucket

import (
	"context"
	"io"
	"sync"

	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/textileio/go-threads/core/thread"
	bucketsClient "github.com/textileio/textile/v2/api/buckets/client"
	bucketsproto "github.com/textileio/textile/v2/api/buckets/pb"
	"github.com/textileio/textile/v2/buckets"
)

type BucketData struct {
	Key       string `json:"_id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	DNSRecord string `json:"dns_record,omitempty"`
	//Archives  Archives `json:"archives"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

type DirEntries bucketsproto.ListPathResponse

type BucketsClient interface {
	PushPath(ctx context.Context, key, pth string, reader io.Reader, opts ...bucketsClient.Option) (result path.Resolved, root path.Resolved, err error)
	PullPath(ctx context.Context, key, pth string, writer io.Writer, opts ...bucketsClient.Option) error
	ListPath(ctx context.Context, key, pth string) (*bucketsproto.ListPathResponse, error)
	RemovePath(ctx context.Context, key, pth string, opts ...bucketsClient.Option) (path.Resolved, error)
	ListIpfsPath(ctx context.Context, ipfsPath path.Path) (*bucketsproto.ListIpfsPathResponse, error)
	PushPathAccessRoles(ctx context.Context, key, path string, roles map[string]buckets.Role) error
}

type EachFunc = func(ctx context.Context, b *Bucket, path string) error

type BucketInterface interface {
	Slug() string
	Key() string
	GetData() BucketData
	GetContext(ctx context.Context) (context.Context, *thread.ID, error)
	GetClient() BucketsClient
	GetThreadID(ctx context.Context) (*thread.ID, error)
	DirExists(ctx context.Context, path string) (bool, error)
	FileExists(ctx context.Context, path string) (bool, error)
	UpdatedAt(ctx context.Context, path string) (int64, error)
	UploadFile(
		ctx context.Context,
		path string,
		reader io.Reader,
	) (result path.Resolved, root path.Path, err error)
	DownloadFile(
		ctx context.Context,
		path string,
		reader io.Reader,
	) (result path.Resolved, root path.Path, err error)
	GetFile(
		ctx context.Context,
		path string,
		w io.Writer,
	) error
	CreateDirectory(
		ctx context.Context,
		path string,
	) (result path.Resolved, root path.Path, err error)
	ListDirectory(
		ctx context.Context,
		path string,
	) (*DirEntries, error)
	DeleteDirOrFile(
		ctx context.Context,
		path string,
	) (path.Resolved, error)
	ItemsCount(
		ctx context.Context,
		path string,
		withRecursive bool,
	) (int32, error)
	Each(
		ctx context.Context,
		path string,
		iterator EachFunc,
		withRecursive bool,
	) (int, error)
}

type Notifier interface {
	OnUploadFile(bucketSlug string, bucketPath string, result path.Resolved, root path.Path)
}

// NOTE: all write operations should use the lock for the bucket to keep consistency
// TODO: Maybe read operations dont need a lock, needs testing
// struct for implementing bucket interface
type Bucket struct {
	lock             sync.RWMutex
	root             *bucketsproto.Root
	bucketsClient    BucketsClient
	getBucketContext GetBucketContextFn
	notifier         Notifier
}

func (b *Bucket) Slug() string {
	return b.GetData().Name
}

type GetBucketContextFn func(context.Context, string) (context.Context, *thread.ID, error)

func New(
	root *bucketsproto.Root,
	getBucketContext GetBucketContextFn,
	bucketsClient BucketsClient,
) *Bucket {
	return &Bucket{
		root:             root,
		bucketsClient:    bucketsClient,
		getBucketContext: getBucketContext,
		notifier:         nil,
	}
}

func (b *Bucket) Key() string {
	return b.GetData().Key
}

func (b *Bucket) GetData() BucketData {
	return BucketData{
		Key:       b.root.Key,
		Name:      b.root.Name,
		Path:      b.root.Path,
		DNSRecord: "",
		CreatedAt: b.root.CreatedAt,
		UpdatedAt: b.root.UpdatedAt,
	}
}

func (b *Bucket) GetContext(ctx context.Context) (context.Context, *thread.ID, error) {
	return b.getBucketContext(ctx, b.root.Name)
}

func (b *Bucket) GetClient() BucketsClient {
	return b.bucketsClient
}

func (b *Bucket) GetThreadID(ctx context.Context) (*thread.ID, error) {
	_, threadID, err := b.GetContext(ctx)
	if err != nil {
		return nil, err
	}

	return threadID, nil
}

func (b *Bucket) AttachNotifier(n Notifier) {
	b.notifier = n
}
