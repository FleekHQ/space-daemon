package watcher

import (
	"os"
)

// isBlacklisted return true if the file or path is not a supported entry
// to trigger file watcher event handler
func isBlacklisted(path string, fileInfo os.FileInfo) bool {
	return fileInfo.Name()[0:1] == "."
}
