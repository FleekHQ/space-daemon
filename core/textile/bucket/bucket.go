package bucket

import (
	"context"
	"io"
	"sync"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/textileio/go-threads/core/thread"
	bucketsClient "github.com/textileio/textile/api/buckets/client"
	bucketsproto "github.com/textileio/textile/api/buckets/pb"
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

type DirEntries bucketsproto.ListPathReply

type BucketsClient interface {
	PushPath(ctx context.Context, key, pth string, reader io.Reader, opts ...bucketsClient.Option) (result path.Resolved, root path.Resolved, err error)
	PullPath(ctx context.Context, key, pth string, writer io.Writer, opts ...bucketsClient.Option) error
	ListPath(ctx context.Context, key, pth string) (*bucketsproto.ListPathReply, error)
	RemovePath(ctx context.Context, key, pth string, opts ...bucketsClient.Option) (path.Resolved, error)
}

// NOTE: all write operations should use the lock for the bucket to keep consistency
// TODO: Maybe read operations dont need a lock, needs testing
// struct for implementing bucket interface
type Bucket struct {
	lock             sync.RWMutex
	root             *bucketsproto.Root
	bucketsClient    BucketsClient
	threadID         *thread.ID
	getBucketContext getBucketContextFn
}

func (b *Bucket) Slug() string {
	return b.GetData().Name
}

type getBucketContextFn func(context.Context, string) (context.Context, *thread.ID, error)

func New(root *bucketsproto.Root, getBucketContext getBucketContextFn, bucketsClient BucketsClient) *Bucket {
	return &Bucket{
		root:             root,
		bucketsClient:    bucketsClient,
		getBucketContext: getBucketContext,
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

func (b *Bucket) getContext(ctx context.Context) (context.Context, *thread.ID, error) {
	return b.getBucketContext(ctx, b.root.Name)
}

func (b *Bucket) MatchInvitesWithMembers(ctx context.Context, invs []string, ms []*domain.Member) (bool, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	// bitmask to make sure duplicates weren't counted
	invsFoundBitmask := 0

	// if the lengths dont match we can already return
	if len(ms) != len(invs) {
		return false, nil
	}

	// loop through each member
	for _, m := range ms {
		// flag to indicate we found member in the invite group
		f2 := false
		for i, inv := range invs {
			if inv == (*m).PublicKey {
				f2 = true
				invsFoundBitmask = invsFoundBitmask | 2 ^ i
				break
			}
		}
		if !f2 {
			// if we get here but f2 was net set, it means we
			// couldnt find the member inside the invites
			return false, nil
		}
	}

	// since it hasnt exited until now,
	// and all the positions have been matched
	// we can return a match
	if invsFoundBitmask == 2^len(invs)-1 {
		return true, nil
	}

	return false, nil
}
