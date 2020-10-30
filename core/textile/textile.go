package textile

import (
	"context"
	"io"

	"github.com/ipfs/go-cid"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/textile/bucket"
	"github.com/FleekHQ/space-daemon/core/textile/model"
	"github.com/FleekHQ/space-daemon/core/textile/sync"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/textileio/go-threads/db"

	buckets_pb "github.com/textileio/textile/v2/api/buckets/pb"
	"github.com/textileio/textile/v2/api/users/client"

	threadsClient "github.com/textileio/go-threads/api/client"
)

const (
	hubTarget                       = "127.0.0.1:3006"
	threadsTarget                   = "127.0.0.1:3006"
	defaultPersonalBucketSlug       = "personal"
	defaultCacheBucketSlug          = "personal_cache"
	defaultPersonalMirrorBucketSlug = "personal_mirror"
	defaultPublicShareBucketSlug    = "personal_public"
)

type BucketRoot buckets_pb.Root

type Bucket interface {
	bucket.BucketInterface
}

type Client interface {
	IsRunning() bool
	IsInitialized() bool
	IsHealthy() bool
	GetDefaultBucket(ctx context.Context) (Bucket, error)
	GetBucket(ctx context.Context, slug string, remoteFile *GetBucketForRemoteFileInput) (Bucket, error)
	GetThreadsConnection() (*threadsClient.Client, error)
	GetModel() model.Model
	ListBuckets(ctx context.Context) ([]Bucket, error)
	ShareBucket(ctx context.Context, bucketSlug string) (*db.Info, error)
	JoinBucket(ctx context.Context, slug string, ti *domain.ThreadInfo) (bool, error)
	CreateBucket(ctx context.Context, bucketSlug string) (Bucket, error)
	ToggleBucketBackup(ctx context.Context, bucketSlug string, bucketBackup bool) (bool, error)
	BucketBackupRestore(ctx context.Context, bucketSlug string) error
	SendMessage(ctx context.Context, recipient crypto.PubKey, body []byte) (*client.Message, error)
	Shutdown() error
	WaitForReady() chan bool
	WaitForHealthy() chan error
	WaitForInitialized() chan bool
	Start(ctx context.Context, cfg config.Config) error
	GetMailAsNotifications(ctx context.Context, seek string, limit int) ([]*domain.Notification, error)
	ShareFilesViaPublicKey(ctx context.Context, paths []domain.FullPath, pubkeys []crypto.PubKey, keys [][]byte) error
	AcceptSharedFilesInvitation(ctx context.Context, invitation domain.Invitation) (domain.Invitation, error)
	RejectSharedFilesInvitation(ctx context.Context, invitation domain.Invitation) (domain.Invitation, error)
	AcceptSharedFileLink(
		ctx context.Context,
		cidHash, password, filename, fileSize string,
	) (*domain.SharedDirEntry, error)
	RemoveKeys(ctx context.Context) error
	AttachMailboxNotifier(notif GrpcMailboxNotifier)
	AttachSynchronizerNotifier(notif sync.EventNotifier)
	GetReceivedFiles(ctx context.Context, accepted bool, seek string, limit int) ([]*domain.SharedDirEntry, string, error)
	GetPublicReceivedFile(ctx context.Context, cidHash string, accepted bool) (*domain.SharedDirEntry, string, error)
	GetPathAccessRoles(ctx context.Context, b Bucket, path string) ([]domain.Member, error)
	GetPublicShareBucket(ctx context.Context) (Bucket, error)
	DownloadPublicGatewayItem(ctx context.Context, cid cid.Cid) (io.ReadCloser, error)
	GetFailedHealthchecks() int
	Listen(ctx context.Context, dbID, threadName string) (<-chan threadsClient.ListenEvent, error)
	RestoreDB(ctx context.Context) error
}

type Buckd interface {
	Stop() error
	Start(ctx context.Context) error
}

type Listener interface {
	Listen(context.Context) error
	Close()
}
