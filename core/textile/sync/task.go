package sync

import (
	"github.com/odeke-em/go-uuid"
)

type taskType string

const (
	addItemTask         taskType = "ADD_ITEM"
	removeItemTask      taskType = "REMOVE_ITEM"
	createBucketTask    taskType = "CREATE_BUCKET"
	pinFileTask         taskType = "PIN_FILE"
	unpinFileTask       taskType = "UNPIN_FILE"
	bucketBackupOnTask  taskType = "TOGGLE_BACKUP_ON"
	bucketBackupOffTask taskType = "TOGGLE_BACKUP_OFF"
)

type taskState string

const (
	taskQueued    taskState = "QUEUED"
	taskPending   taskState = "PENDING"
	taskSucceeded taskState = "SUCCESS"
	taskFailed    taskState = "FAILED"
	taskDequeued  taskState = "DEQUEUED"
)

type Task struct {
	ID             string    `json:"id"`
	State          taskState `json:"state"`
	Type           taskType  `json:"type"`
	Args           []string  `json:"args"`
	Parallelizable bool      `json:"parallelizable"`

	// Set to -1 for infinite retries
	MaxRetries int `json:"maxRetries"`
	Retries    int `json:"retries"`
}

func newTask(t taskType, args []string) *Task {
	id := uuid.UUID1().String()

	return &Task{
		ID:             id,
		State:          taskQueued,
		Type:           t,
		Args:           args,
		Parallelizable: false,
		MaxRetries:     -1,
		Retries:        0,
	}
}
