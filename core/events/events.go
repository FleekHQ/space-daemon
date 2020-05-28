package events


// These file defines events that daemon can propagate through all layers
type FileEvent struct {
	Path string
}


func NewFileEvent(path string) FileEvent {
	return FileEvent{
		Path: path,
	}
}
