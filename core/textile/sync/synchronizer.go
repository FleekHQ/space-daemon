package sync

import (
	"container/list"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	s "sync"
	"time"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile/bucket"
	"github.com/FleekHQ/space-daemon/core/textile/hub"
	"github.com/FleekHQ/space-daemon/core/textile/model"
	"github.com/FleekHQ/space-daemon/log"
	threadsClient "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	nc "github.com/textileio/go-threads/net/api/client"
	bucketsClient "github.com/textileio/textile/api/buckets/client"
)

type GetMirrorBucketFn func(ctx context.Context, slug string) (bucket.BucketInterface, error)
type GetBucketFn func(ctx context.Context, slug string) (bucket.BucketInterface, error)
type GetBucketCtxFn func(ctx context.Context, sDbID string, bucketSlug string, ishub bool, enckey []byte) (context.Context, *thread.ID, error)

type synchronizer struct {
	taskQueue        *list.List
	filePinningQueue *list.List
	queueHashMap     map[string]*Task
	st               store.Store
	model            model.Model
	syncNeeded       chan (bool)
	shuttingDown     chan (bool)
	queueMutexMap    map[*list.List]*s.Mutex
	getMirrorBucket  GetMirrorBucketFn
	getBucket        GetBucketFn
	getBucketCtx     GetBucketCtxFn
	kc               keychain.Keychain
	hubAuth          hub.HubAuth
	hubBuckets       *bucketsClient.Client
	hubThreads       *threadsClient.Client
	cfg              config.Config
	netc             *nc.Client
}

// Creates a new Synchronizer
func New(
	st store.Store,
	model model.Model,
	kc keychain.Keychain,
	hubAuth hub.HubAuth,
	hb *bucketsClient.Client,
	ht *threadsClient.Client,
	netc *nc.Client,
	cfg config.Config,
	getMirrorBucket GetMirrorBucketFn,
	getBucket GetBucketFn,
	getBucketCtx GetBucketCtxFn,
) *synchronizer {
	taskQueue := list.New()
	filePinningQueue := list.New()

	queueMutexMap := make(map[*list.List]*s.Mutex)
	queueMutexMap[taskQueue] = &s.Mutex{}
	queueMutexMap[filePinningQueue] = &s.Mutex{}

	return &synchronizer{
		taskQueue:        taskQueue,
		filePinningQueue: filePinningQueue,
		queueHashMap:     make(map[string]*Task),
		st:               st,
		model:            model,
		syncNeeded:       make(chan bool),
		shuttingDown:     make(chan bool),
		queueMutexMap:    queueMutexMap,
		getMirrorBucket:  getMirrorBucket,
		getBucket:        getBucket,
		getBucketCtx:     getBucketCtx,
		kc:               kc,
		hubAuth:          hubAuth,
		hubBuckets:       hb,
		hubThreads:       ht,
		cfg:              cfg,
		netc:             netc,
	}
}

// Notify Textile synchronizer that an add item operation needs to be synced
func (s *synchronizer) NotifyItemAdded(bucket, path string) {
	t := newTask(addItemTask, []string{bucket, path})
	s.enqueueTask(t, s.taskQueue)

	pft := newTask(pinFileTask, []string{bucket, path})
	pft.Parallelizable = true
	s.enqueueTask(pft, s.filePinningQueue)

	s.syncNeeded <- true
}

// Notify Textile synchronizer that a remove item operation needs to be synced
func (s *synchronizer) NotifyItemRemoved(bucket, path string) {
	t := newTask(removeItemTask, []string{bucket, path})
	s.enqueueTask(t, s.taskQueue)

	uft := newTask(unpinFileTask, []string{bucket, path})
	uft.Parallelizable = true
	s.enqueueTask(uft, s.filePinningQueue)

	s.syncNeeded <- true
}

func (s *synchronizer) NotifyBucketCreated(bucket string, enckey []byte) {
	t := newTask(createBucketTask, []string{bucket, hex.EncodeToString(enckey)})
	s.enqueueTask(t, s.taskQueue)
}

func (s *synchronizer) NotifyBucketBackupOn(bucket string) {
	t := newTask(bucketBackupOnTask, []string{bucket})
	s.enqueueTask(t, s.taskQueue)
}

func (s *synchronizer) NotifyBucketBackupOff(bucket string) {
	t := newTask(bucketBackupOffTask, []string{bucket})
	s.enqueueTask(t, s.taskQueue)
}

// Starts the synchronizer, which will constantly be checking if there are syncing tasks pending
func (s *synchronizer) Start(ctx context.Context) {
	// Sync loop
	go func() {
		s.startSyncLoop(ctx, s.taskQueue)
	}()
	go func() {
		s.startSyncLoop(ctx, s.filePinningQueue)
	}()
}

// Restores a previously initialized queue
func (s *synchronizer) RestoreQueue() error {
	if err := s.restoreQueue(); err != nil {
		return err
	}

	return nil
}

func (s *synchronizer) startSyncLoop(ctx context.Context, queue *list.List) {
	queueMutex := s.queueMutexMap[queue]
	// Initial sync
	queueMutex.Lock()
	s.sync(ctx, queue)
	queueMutex.Unlock()

	for {
		queueMutex.Lock()
		timeAfterNextSync := 30 * time.Second

		select {
		case <-time.After(timeAfterNextSync):
			s.sync(ctx, queue)

		case <-s.syncNeeded:
			s.sync(ctx, queue)

		// Break execution in case of shutdown
		case <-ctx.Done():
			queueMutex.Unlock()
			return
		case <-s.shuttingDown:
			queueMutex.Unlock()
			return
		}

		queueMutex.Unlock()
	}
}

func (s *synchronizer) Shutdown() {
	s.shuttingDown <- true
}

var errMaxRetriesSurpassed = errors.New("max retries surpassed")

func (s *synchronizer) executeTask(ctx context.Context, t *Task) error {
	t.State = taskPending
	var err error

	switch t.Type {
	case addItemTask:
		err = s.processAddItem(ctx, t)
	case removeItemTask:
		err = s.processRemoveItem(ctx, t)
	case pinFileTask:
		err = s.processPinFile(ctx, t)
	case unpinFileTask:
		err = s.processUnpinFile(ctx, t)
	case createBucketTask:
		err = s.processCreateBucket(ctx, t)
	case bucketBackupOnTask:
		err = s.processBucketBackupOn(ctx, t)
	case bucketBackupOffTask:
		err = s.processBucketBackupOff(ctx, t)
	default:
		log.Warn("Unexpected action on Textile sync, executeTask")
	}

	if err != nil {
		t.State = taskFailed
		t.Retries++

		// Remove from queue if it surpassed the max amount of retries
		if t.MaxRetries != -1 && t.Retries > t.MaxRetries {
			return errMaxRetriesSurpassed
		}

		// Retry task
		t.State = taskQueued
	}

	return err
}

func (s *synchronizer) sync(ctx context.Context, queue *list.List) error {
	queueName := "buckets"
	if queue == s.filePinningQueue {
		queueName = "file pinning"
	}

	log.Debug(fmt.Sprintf("Textile sync [%s]: Sync start", queueName))
	if queue.Len() == 0 {
		log.Debug(fmt.Sprintf("Textile sync [%s]: empty queue", queueName))
	}

	curr := queue.Front()

	for curr != nil {
		task := curr.Value.(*Task)

		log.Debug(fmt.Sprintf("Textile sync [%s]: Processing task %s", queueName, task.Type))
		if task.State != taskQueued {
			// If task is already in process or finished, skip
			continue
		}

		handleExecResult := func(queueEl *list.Element, err error) {

			if err == errMaxRetriesSurpassed {
				queue.Remove(queueEl)
			}

			if err == nil {
				// Task completed successfully
				log.Debug(fmt.Sprintf("Textile sync [%s]: task completed succesfully", queueName))
				queue.Remove(queueEl)
			} else {
				log.Error(fmt.Sprintf("Textile sync [%s]: task failed", queueName), err)
			}

			if err := s.storeQueue(); err != nil {
				log.Error("Error while storing Textile task queue state", err)
			}
		}

		if task.Parallelizable {
			// Creating aux var in case the go func gets ran after curr = curr.Next()
			queueEl := curr

			go func() {
				err := s.executeTask(ctx, task)
				handleExecResult(queueEl, err)
			}()
		} else {
			err := s.executeTask(ctx, task)
			handleExecResult(curr, err)

			if err != nil {
				// Break from the loop (avoid executing succeeding tasks)
				return err
			}
		}

		curr = curr.Next()
	}

	log.Debug(fmt.Sprintf("Textile sync [%s]: Sync end", queueName))

	return nil
}
