package sync

import (
	"container/list"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
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

const maxParallelTasks = 16

type synchronizer struct {
	taskQueue        *list.List
	filePinningQueue *list.List
	queueHashMap     map[string]*Task
	st               store.Store
	model            model.Model
	syncNeeded       chan (bool)
	shuttingDownMap  map[*list.List]chan (bool)
	queueMutexMap    map[*list.List]*sync.Mutex
	getMirrorBucket  GetMirrorBucketFn
	getBucket        GetBucketFn
	getBucketCtx     GetBucketCtxFn
	kc               keychain.Keychain
	hubAuth          hub.HubAuth
	hubBuckets       *bucketsClient.Client
	hubThreads       *threadsClient.Client
	cfg              config.Config
	netc             *nc.Client
	queueWg          sync.WaitGroup
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

	queueMutexMap := make(map[*list.List]*sync.Mutex)
	queueMutexMap[taskQueue] = &sync.Mutex{}
	queueMutexMap[filePinningQueue] = &sync.Mutex{}

	shuttingDownMap := make(map[*list.List]chan bool)
	shuttingDownMap[taskQueue] = make(chan bool)
	shuttingDownMap[filePinningQueue] = make(chan bool)

	queueWg := sync.WaitGroup{}

	return &synchronizer{
		taskQueue:        taskQueue,
		filePinningQueue: filePinningQueue,
		queueHashMap:     make(map[string]*Task),
		st:               st,
		model:            model,
		syncNeeded:       make(chan bool),
		shuttingDownMap:  shuttingDownMap,
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
		queueWg:          queueWg,
	}
}

// Notify Textile synchronizer that an add item operation needs to be synced
func (s *synchronizer) NotifyItemAdded(bucket, path string) {
	t := newTask(addItemTask, []string{bucket, path})
	s.enqueueTask(t, s.taskQueue)

	pft := newTask(pinFileTask, []string{bucket, path})
	pft.Parallelizable = true
	s.enqueueTask(pft, s.filePinningQueue)

	s.notifySyncNeeded()
}

// Notify Textile synchronizer that a remove item operation needs to be synced
func (s *synchronizer) NotifyItemRemoved(bucket, path string) {
	t := newTask(removeItemTask, []string{bucket, path})
	s.enqueueTask(t, s.taskQueue)

	uft := newTask(unpinFileTask, []string{bucket, path})
	uft.Parallelizable = true
	s.enqueueTask(uft, s.filePinningQueue)

	s.notifySyncNeeded()
}

func (s *synchronizer) NotifyBucketCreated(bucket string, enckey []byte) {
	t := newTask(createBucketTask, []string{bucket, hex.EncodeToString(enckey)})
	s.enqueueTask(t, s.taskQueue)
	s.notifySyncNeeded()
}

func (s *synchronizer) NotifyBucketBackupOn(bucket string) {
	t := newTask(bucketBackupOnTask, []string{bucket})
	s.enqueueTask(t, s.taskQueue)

	s.notifySyncNeeded()
}

func (s *synchronizer) NotifyBucketBackupOff(bucket string) {
	t := newTask(bucketBackupOffTask, []string{bucket})
	s.enqueueTask(t, s.taskQueue)

	s.notifySyncNeeded()
}

func (s *synchronizer) notifySyncNeeded() {
	select {
	case s.syncNeeded <- true:
	default:
	}
}

// Starts the synchronizer, which will constantly be checking if there are syncing tasks pending
func (s *synchronizer) Start(ctx context.Context) {
	s.queueWg.Add(2)
	// Sync loop
	go func() {
		s.startSyncLoop(ctx, s.taskQueue)
		s.queueWg.Done()
	}()
	go func() {
		s.startSyncLoop(ctx, s.filePinningQueue)
		s.queueWg.Done()
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

Loop:
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
			s.Shutdown()
			break Loop
		case <-s.shuttingDownMap[queue]:
			queueMutex.Unlock()
			break Loop
		}

		queueMutex.Unlock()
	}
}

func (s *synchronizer) Shutdown() {
	s.shuttingDownMap[s.taskQueue] <- true
	s.shuttingDownMap[s.filePinningQueue] <- true
	s.queueWg.Wait()
}

func (s *synchronizer) String() string {
	queues := []*list.List{s.filePinningQueue, s.taskQueue}

	res := ""
	for _, q := range queues {
		res = res + s.queueString(q) + "\n"
	}

	return res
}

var errMaxRetriesSurpassed = errors.New("max retries surpassed")

func (s *synchronizer) executeTask(ctx context.Context, t *Task) error {
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
			t.State = taskDequeued
			return errMaxRetriesSurpassed
		}

		// Retry task
		t.State = taskQueued
	} else {
		t.State = taskSucceeded
	}

	return err
}

func (s *synchronizer) sync(ctx context.Context, queue *list.List) error {
	queueName := "buckets"
	if queue == s.filePinningQueue {
		queueName = "file pinning"
	}

	log.Debug(fmt.Sprintf("Textile sync [%s]: Sync start", queueName))
	log.Debug(s.queueString(queue))

	parallelTaskCount := 0
	ptWg := sync.WaitGroup{}

	for curr := queue.Front(); curr != nil; curr = curr.Next() {
		task := curr.Value.(*Task)

		if task.State != taskQueued {
			// If task is already in process or finished, skip
			continue
		}
		log.Debug(fmt.Sprintf("Textile sync [%s]: Processing task %s", queueName, task.Type))
		task.State = taskPending

		handleExecResult := func(err error) {
			if err == nil {
				// Task completed successfully
				log.Debug(fmt.Sprintf("Textile sync [%s]: task completed succesfully", queueName))
			} else {
				log.Error(fmt.Sprintf("Textile sync [%s]: task failed", queueName), err)
			}
		}

		if task.Parallelizable && parallelTaskCount < maxParallelTasks {
			parallelTaskCount++
			ptWg.Add(1)

			go func() {
				err := s.executeTask(ctx, task)
				handleExecResult(err)
				parallelTaskCount--
				ptWg.Done()
			}()
		} else {
			err := s.executeTask(ctx, task)
			handleExecResult(err)

			if err != nil {
				// Break from the loop (avoid executing next tasks)
				return err
			}
		}
	}

	// Remove successful and dequeued tasks from queue
	curr := queue.Front()
	for curr != nil {
		task := curr.Value.(*Task)
		next := curr.Next()

		switch task.State {
		case taskDequeued:
			queue.Remove(curr)
		case taskSucceeded:
			queue.Remove(curr)
		default:
		}

		curr = next
	}

	ptWg.Wait()

	if err := s.storeQueue(); err != nil {
		log.Error("Error while storing Textile task queue state", err)
	}

	log.Debug(fmt.Sprintf("Textile sync [%s]: Sync end", queueName))

	return nil
}
