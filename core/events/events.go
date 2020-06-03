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
