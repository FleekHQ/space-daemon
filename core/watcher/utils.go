package watcher

import (
	"os"
	//"golang.org/x/sys/windows"
)

// isBlacklisted return true if the file or path is not a supported entry
// to trigger file watcher event handler
func isBlacklisted(path string, fileInfo os.FileInfo) bool {
	// QQ: Do we want to just ignore all hidden files?
	//if runtime.GOOS == "windows" {
	//	pointer, err := windows.UTF16PtrFromString(path)
	//	if err != nil {
	//		return false
	//	}
	//	attributes, err := windows.GetFileAttributes(pointer)
	//	if err != nil {
	//		return false
	//	}
	//	return attributes&windows.FILE_ATTRIBUTE_HIDDEN != 0
	//} else if fileInfo.Name()[0:1] == "." {
	//	return true
	//}

	if fileInfo.Name() == ".DS_Store" {
		return true
	}
	// TODO: Handle Windows and Linux platforms

	return false
}
