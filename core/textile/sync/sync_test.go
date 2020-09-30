package sync_test

import (
	"context"
	"errors"
	sy "sync"
	"testing"

	"github.com/FleekHQ/space-daemon/core/textile"
	"github.com/FleekHQ/space-daemon/core/textile/bucket"
	"github.com/FleekHQ/space-daemon/core/textile/sync"
	"github.com/FleekHQ/space-daemon/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/textileio/go-threads/core/thread"
)

var (
	mockStore      *mocks.Store
	mockClient     *mocks.Client
	mockModel      *mocks.Model
	mockKeychain   *mocks.Keychain
	mockHubAuth    *mocks.HubAuth
	mockCfg        *mocks.Config
	mockRemoteFile = &textile.GetBucketForRemoteFileInput{
		Bucket: "",
		DbID:   "",
		Path:   "",
	}
)

func initSync(t *testing.T) sync.Synchronizer {
	mockStore = new(mocks.Store)
	mockModel = new(mocks.Model)
	mockKeychain = new(mocks.Keychain)
	mockHubAuth = new(mocks.HubAuth)
	mockCfg = new(mocks.Config)
	mockClient = new(mocks.Client)

	mockStore.On("IsOpen").Return(true)

	getLocalBucketFn := func(ctx context.Context, slug string) (bucket.BucketInterface, error) {
		return mockClient.GetBucket(ctx, slug, nil)
	}

	getMirrorBucketFn := func(ctx context.Context, slug string) (bucket.BucketInterface, error) {
		return mockClient.GetBucket(ctx, slug, mockRemoteFile)
	}

	getBucketCtxFn := func(ctx context.Context, sDbID string, bucketSlug string, ishub bool, enckey []byte) (context.Context, *thread.ID, error) {
		return ctx, nil, nil
	}

	s := sync.New(mockStore, mockModel, mockKeychain, mockHubAuth, nil, nil, nil, mockCfg, getMirrorBucketFn, getLocalBucketFn, getBucketCtxFn)

	return s
}

var mutex = &sy.Mutex{}

func TestSync_ProcessTask(t *testing.T) {
	mutex.Lock()
	defer mutex.Unlock()

	s := initSync(t)
	ctx := context.Background()

	s.NotifyItemAdded("Bucket", "path")

	// Makes the processAddItem and processPinFilefail right away
	mockModel.On("FindBucket", mock.Anything, mock.Anything).Return(nil, errors.New("some error"))
	mockClient.On("GetBucket", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("some error"))

	mockStore.On("Set", mock.Anything, mock.Anything).Return(nil)

	s.Start(ctx)

	s.Shutdown()

	expectedState := "Textile sync [file pinning]: Total: 1, Queued: 1, Pending: 0, Failed: 0\nTextile sync [buckets]: Total: 1, Queued: 1, Pending: 0, Failed: 0\n"

	assert.Equal(t, expectedState, s.String())
	mockModel.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestSync_Restore(t *testing.T) {
	mutex.Lock()
	defer mutex.Unlock()

	s := initSync(t)
	ctx := context.Background()

	s.NotifyItemAdded("Bucket", "path")

	// Makes the processAddItem and processPinFilefail right away
	mockModel.On("FindBucket", mock.Anything, mock.Anything).Return(nil, errors.New("some error"))
	mockClient.On("GetBucket", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("some error"))

	mockStore.On("Set", []byte(sync.QueueStoreKey), mock.Anything).Return(nil)

	s.Start(ctx)

	s.Shutdown()

	ogMockStore := mockStore

	s2 := initSync(t)

	// Make Store.Get return the data set previously
	storeArgs := ogMockStore.Calls[0].Arguments
	bytes := storeArgs.Get(1)
	mockStore.On("Get", []byte(sync.QueueStoreKey)).Return(bytes, nil)

	err := s2.RestoreQueue()

	mockModel.On("FindBucket", mock.Anything, mock.Anything).Return(nil, errors.New("some error"))
	mockClient.On("GetBucket", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("some error"))
	mockStore.On("Set", []byte(sync.QueueStoreKey), mock.Anything).Return(nil)

	// Note we are not calling NotifyItemAdded therefore the state must be picked from the Restore func
	s2.Start(ctx)

	s2.Shutdown()

	assert.Nil(t, err)
	assert.Equal(t, s.String(), s2.String())
}
