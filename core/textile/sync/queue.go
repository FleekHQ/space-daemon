package sync

import (
	"container/list"
	"encoding/json"
	"fmt"
)

const QueueStoreKey = "TextileSyncTaskQueue"

type marshalledQueue struct {
	QueueAsSlice     []Task `json:"queueAsSlice"`
	FileQueueAsSlice []Task `json:"fileQueueAsSlice"`
}

func (s *synchronizer) enqueueTask(task *Task, queue *list.List) {
	if s.isTaskEnqueued(task) == false {
		queue.PushBack(task)
		s.queueHashMap[task.ID] = task
	}
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
	queueAsSlice := []Task{}
	currEl := s.taskQueue.Front()

	for currEl != nil {
		queueAsSlice = append(queueAsSlice, *currEl.Value.(*Task))
		currEl = currEl.Next()
	}

	// Store file pinning queue
	fileQueueAsSlice := []Task{}
	currEl = s.filePinningQueue.Front()

	for currEl != nil {
		fileQueueAsSlice = append(fileQueueAsSlice, *currEl.Value.(*Task))
		currEl = currEl.Next()
	}

	objToMarshal := &marshalledQueue{
		QueueAsSlice:     queueAsSlice,
		FileQueueAsSlice: fileQueueAsSlice,
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

	for _, el := range queue.QueueAsSlice {
		s.enqueueTask(&el, s.taskQueue)
	}

	for _, el := range queue.FileQueueAsSlice {
		s.enqueueTask(&el, s.filePinningQueue)
	}

	return nil
}

func (s *synchronizer) isTaskEnqueued(task *Task) bool {
	existingTask := s.queueHashMap[task.ID]
	if existingTask == nil {
		return false
	}

	isPending := existingTask.State == taskQueued || existingTask.State == taskPending
	if isPending {
		return true
	}

	return false
}

func (s *synchronizer) queueString(queue *list.List) string {
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

	return fmt.Sprintf("Textile sync [%s]: Total: %d, Queued: %d, Pending: %d, Failed: %d", queueName, queue.Len(), queued, pending, failed)
}
