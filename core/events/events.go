package events

import (
	"os"
)

// These file defines events that daemon can propagate through all layers

type FileEventType string

const (
	FileAdded   FileEventType = "FileAdded"
	FileDeleted FileEventType = "FileDeleted"
	FileUpdated FileEventType = "FileUpdated"

	FolderAdded   FileEventType = "FolderAdded"
	FolderDeleted FileEventType = "FolderDeleted"
	// NOTE: not sure if this needs to be specific to rename or copy
	FolderUpdated FileEventType = "FolderUpdated"
)

type FileEvent struct {
	Path string
	Info os.FileInfo
	Type FileEventType
}

func NewFileEvent(path string, eventType FileEventType, info os.FileInfo) FileEvent {
	return FileEvent{
		Path: path,
		Type: eventType,
		Info: info,
	}
}

type TextileEvent struct {
	BucketName string
}

func NewTextileEvent(bucketname string) TextileEvent {
	return TextileEvent{
		BucketName: bucketname,
	}
}

type InvitationStatus int

const (
	Pending InvitationStatus = 0
	Accepted
	Rejected
)

type NotificationType int

const (
	InvitationType NotificationType = 0
)

type Invitation struct {
	InviterPublicKey string
	InvitationID     string
	Status           InvitationStatus
	ItemPaths        []string
}

type NotificationEvent struct {
	Subject       string
	Body          string
	RelatedObject interface{}
	Type          NotificationType
	CreatedAt     int64
	ReadAt        int64
}
