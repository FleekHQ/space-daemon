package textile

import (
	"context"
	"io"

	"github.com/ipfs/go-cid"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/textile/bucket"
	"github.com/FleekHQ/space-daemon/core/textile/model"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"

	buckets_pb "github.com/textileio/textile/api/buckets/pb"
	"github.com/textileio/textile/api/users/client"

	threadsClient "github.com/textileio/go-threads/api/client"
)

const (
	hubTarget                       = "127.0.0.1:3006"
	threadsTarget                   = "127.0.0.1:3006"
	defaultPersonalBucketSlug       = "personal"
	defaultPersonalMirrorBucketSlug = "personal_mirror"
	defaultPublicShareBucketSlug    = "personal_public"
)

type BucketRoot buckets_pb.Root

type Bucket interface {
	Slug() string
	Key() string
	GetData() bucket.BucketData
	GetThreadID(ctx context.Context) (*thread.ID, error)
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
}

type Client interface {
	IsRunning() bool
	GetDefaultBucket(ctx context.Context) (Bucket, error)
	GetBucket(ctx context.Context, slug string, remoteFile *GetBucketForRemoteFileInput) (Bucket, error)
	GetThreadsConnection() (*threadsClient.Client, error)
	GetModel() model.Model
	ListBuckets(ctx context.Context) ([]Bucket, error)
	ShareBucket(ctx context.Context, bucketSlug string) (*db.Info, error)
	JoinBucket(ctx context.Context, slug string, ti *domain.ThreadInfo) (bool, error)
	CreateBucket(ctx context.Context, bucketSlug string) (Bucket, error)
	ToggleBucketBackup(ctx context.Context, bucketSlug string, bucketBackup bool) (bool, error)
	SendMessage(ctx context.Context, recipient crypto.PubKey, body []byte) (*client.Message, error)
	Shutdown() error
	WaitForReady() chan bool
	WaitForHealthy() chan bool
	Start(ctx context.Context, cfg config.Config) error
	GetMailAsNotifications(ctx context.Context, seek string, limit int) ([]*domain.Notification, error)
	ShareFilesViaPublicKey(ctx context.Context, paths []domain.FullPath, pubkeys []crypto.PubKey, keys [][]byte) error
	AcceptSharedFilesInvitation(ctx context.Context, invitation domain.Invitation) (domain.Invitation, error)
	RejectSharedFilesInvitation(ctx context.Context, invitation domain.Invitation) (domain.Invitation, error)
	RemoveKeys() error
	AttachMailboxNotifier(notif GrpcMailboxNotifier)
	IsBucketBackup(ctx context.Context, bucketSlug string) bool
	IsMirrorFile(ctx context.Context, path, bucketSlug string) bool
	UploadFileToHub(ctx context.Context, b Bucket, path string, reader io.Reader) (result path.Resolved, root path.Path, err error)
	MarkMirrorFileBackup(ctx context.Context, path, bucketSlug string) (*domain.MirrorFile, error)
	GetReceivedFiles(ctx context.Context, accepted bool, seek string, limit int) ([]*domain.SharedDirEntry, string, error)
	GetPathAccessRoles(ctx context.Context, b Bucket, bucketKey, path string) ([]string, error)
	GetPublicShareBucket(ctx context.Context) (Bucket, error)
	DownloadPublicGatewayItem(ctx context.Context, cid cid.Cid) (io.ReadCloser, error)
}

type Buckd interface {
	Stop() error
	Start(ctx context.Context) error
}
