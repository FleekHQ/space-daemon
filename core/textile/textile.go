package textile

import (
	"context"
	"io"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/textile/bucket"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/libp2p/go-libp2p-core/crypto"
	tc "github.com/textileio/go-threads/api/client"

	buckets_pb "github.com/textileio/textile/api/buckets/pb"
	"github.com/textileio/textile/api/users/client"

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
	UploadFileToHub(
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
}

type Client interface {
	IsRunning() bool
	GetDefaultBucket(ctx context.Context) (Bucket, error)
	GetBucket(ctx context.Context, slug string) (Bucket, error)
	GetThreadsConnection() (*threadsClient.Client, error)
	ListBuckets(ctx context.Context) ([]Bucket, error)
	ShareBucket(ctx context.Context, bucketSlug string) (*tc.DBInfo, error)
	JoinBucket(ctx context.Context, slug string, ti *domain.ThreadInfo) (bool, error)
	CreateBucket(ctx context.Context, bucketSlug string) (Bucket, error)
	ToggleBucketBackup(ctx context.Context, bucketSlug string, bucketBackup bool) (bool, error)
	IsBucketBackup(ctx context.Context, bucketSlug string) bool
	IsMirrorFile(ctx context.Context, path, bucketSlug string) bool
	BackupFile(ctx context.Context, path, bucketSlug string) (*MirrorFile, error)
	SendMessage(ctx context.Context, recipient crypto.PubKey, body []byte) (*client.Message, error)
	Shutdown() error
	WaitForReady() chan bool
	Start(ctx context.Context, cfg config.Config) error
	ShareFilesViaPublicKey(ctx context.Context, bucketName string, paths []string, pubkeys []crypto.PubKey) error
	RemoveKeys()
}

type Buckd interface {
	Stop() error
	Start(ctx context.Context) error
}
