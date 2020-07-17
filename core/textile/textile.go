package textile

import (
	"context"
	"io"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/textile/bucket"
	"github.com/ipfs/interface-go-ipfs-core/path"
	tc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"

	buckets_pb "github.com/textileio/textile/api/buckets/pb"

	threadsClient "github.com/textileio/go-threads/api/client"
)

const (
	hubTarget                 = "127.0.0.1:3006"
	threadsTarget             = "127.0.0.1:3006"
	threadIDStoreKey          = "thread_id"
	defaultPersonalBucketSlug = "personal"
)

type BucketRoot buckets_pb.Root

type Bucket interface {
	Slug() string
	Key() string
	GetData() bucket.BucketData
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
	) (*bucket.DirEntries, error)
	DeleteDirOrFile(
		ctx context.Context,
		path string,
	) (path.Resolved, error)
	MatchInvitesWithMembers(ctx context.Context, invs []domain.Invitation, ms []domain.Member) (bool, error)
}

type Client interface {
	IsRunning() bool
	GetDefaultBucket(ctx context.Context) (Bucket, error)
	GetBucket(ctx context.Context, slug string) (Bucket, error)
	GetThreadsConnection() (*threadsClient.Client, error)
	GetBucketContext(ctx context.Context, bucketSlug string) (context.Context, *thread.ID, error)
	ListBuckets(ctx context.Context) ([]Bucket, error)
	ShareBucket(ctx context.Context, bucketSlug string) (*tc.DBInfo, error)
	JoinBucket(ctx context.Context, slug string, ti *domain.ThreadInfo) (bool, error)
	CreateBucket(ctx context.Context, bucketSlug string) (Bucket, error)
	Shutdown() error
	WaitForReady() chan bool
	Start(ctx context.Context, cfg config.Config) error
	FindBucketWithMembers(ctx context.Context, invs []domain.Invitation) (Bucket, error)
	CopyItems(ctx context.Context, srcBucket string, paths []string, trgBucket string) error
	SetMembers(ctx context.Context, slug string, ms []domain.Member) error
	GetMembers(ctx context.Context, slug string) ([]domain.Member, error)
}

type Buckd interface {
	Stop() error
	Start(ctx context.Context) error
}
