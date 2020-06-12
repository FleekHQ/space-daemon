package textile

import (
	"context"
	"io"

	"github.com/FleekHQ/space/config"
	"github.com/ipfs/interface-go-ipfs-core/path"

	buckets_pb "github.com/textileio/textile/api/buckets/pb"

	threadsClient "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
)

const (
	hubTarget                 = "127.0.0.1:3006"
	threadsTarget             = "127.0.0.1:3006"
	threadIDStoreKey          = "thread_id"
	defaultPersonalBucketSlug = "personal"
)

type BucketRoot buckets_pb.Root
type DirEntries buckets_pb.ListPathReply

type BucketData struct {
	Key       string `json:"_id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	DNSRecord string `json:"dns_record,omitempty"`
	//Archives  Archives `json:"archives"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

type Bucket interface {
	Slug() string
	Key() string
	GetData() BucketData
	GetContext(ctx context.Context) (context.Context, *thread.ID, error)
	DirExists(ctx context.Context, path string) (bool, error)
	FileExists(ctx context.Context, path string) (bool, error)
	UploadFile(
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
}

type Client interface {
	IsRunning() bool
	GetDefaultBucket(ctx context.Context) (Bucket, error)
	GetBucket(ctx context.Context, slug string) (Bucket, error)
	GetBaseThreadsContext(ctx context.Context) (context.Context, error)
	GetBucketContext(ctx context.Context, bucketSlug string) (context.Context, *thread.ID, error)
	GetThreadsConnection() (*threadsClient.Client, error)
	ListBuckets(ctx context.Context) ([]Bucket, error)
	CreateBucket(ctx context.Context, bucketSlug string) (Bucket, error)
	Stop() error
	WaitForReady() chan bool
	StartAndBootstrap(ctx context.Context, cfg config.Config) error
}
