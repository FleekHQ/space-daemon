package sync

import (
	"container/list"
	"encoding/json"
	"fmt"

	"github.com/FleekHQ/space-daemon/log"
)

const QueueStoreKey = "TextileSyncTaskQueue"

type marshalledQueue struct {
	queueAsSlice     []*Task          `json:"queueAsSlice"`
	fileQueueAsSlice []*Task          `json:"fileQueueAsSlice"`
	hashMap          map[string]*Task `json:"queueHashMap"`
}

func (s *synchronizer) enqueueTask(task *Task, queue *list.List) {
	queue.PushBack(task)
	s.queueHashMap[task.ID] = task
}

func (s *synchronizer) dequeueTask(queue *list.List) *Task {
	queueItem := queue.Front()
	s.taskQueue.Remove(queueItem)

	task := queueItem.Value.(*Task)
	delete(s.queueHashMap, task.ID)

	return task
}

func (s *synchronizer) storeQueue() error {
	// Store main queue
	queueAsSlice := []*Task{}
	currEl := s.taskQueue.Front()

	for currEl != nil {
		queueAsSlice = append(queueAsSlice, currEl.Value.(*Task))
		currEl = currEl.Next()
	}

	// Store file pinning queue
	fileQueueAsSlice := []*Task{}
	currEl = s.filePinningQueue.Front()

	for currEl != nil {
		fileQueueAsSlice = append(fileQueueAsSlice, currEl.Value.(*Task))
		currEl = currEl.Next()
	}

	objToMarshal := &marshalledQueue{
		queueAsSlice:     queueAsSlice,
		fileQueueAsSlice: fileQueueAsSlice,
		hashMap:          s.queueHashMap,
	}

	marshalled, err := json.Marshal(objToMarshal)
	if err != nil {
		return err
	}

	err = s.st.Set([]byte(QueueStoreKey), marshalled)
	if err != nil {
		return err
	}

	return nil
}

func (s *synchronizer) restoreQueue() error {
	queueMutex1 := s.queueMutexMap[s.taskQueue]
	queueMutex2 := s.queueMutexMap[s.filePinningQueue]
	queueMutex1.Lock()
	queueMutex2.Lock()
	defer queueMutex1.Unlock()
	defer queueMutex2.Unlock()

	data, err := s.st.Get([]byte(QueueStoreKey))
	if err != nil {
		return err
	}

	queue := &marshalledQueue{}
	err = json.Unmarshal(data, queue)
	if err != nil {
		return err
	}

	for _, el := range queue.queueAsSlice {
		s.enqueueTask(el, s.taskQueue)
	}

	for _, el := range queue.fileQueueAsSlice {
		s.enqueueTask(el, s.filePinningQueue)
	}

	return nil
}

func (s *synchronizer) isTaskEnqueued(task *Task) bool {
	if s.queueHashMap[task.ID] != nil {
		return true
	}

	return false
}

func (s *synchronizer) printQueueStats(queue *list.List) {
	queueName := "buckets"
	if queue == s.filePinningQueue {
		queueName = "file pinning"
	}

	failed, queued, pending := 0, 0, 0

	for curr := queue.Front(); curr != nil; curr = curr.Next() {
		task := curr.Value.(*Task)

		switch task.State {
		case taskPending:
			pending++
		case taskFailed:
			failed++
		case taskQueued:
			queued++
		}
	}

	log.Debug(fmt.Sprintf("Textile sync [%s]: Total: %d, Queued: %d, Pending: %d, Failed: %d", queueName, queue.Len(), queued, pending, failed))
}
