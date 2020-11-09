package util

import (
	"os"
	s "strings"

	"github.com/mitchellh/go-homedir"
)

// ResolvePath resolves a path into its full qualified path
// alias like `~` is  expanded based on the current user
func ResolvePath(path string) (string, error) {
	fullPath := path
	if home, err := homedir.Dir(); err == nil {
		// If the path contains ~, we replace it with the actual home directory
		fullPath = s.Replace(path, "~", home, -1)
	} else {
		return "", err
	}

	return fullPath, nil
}

// DirEntryExists check if the file or directory with the given path exits.
func DirEntryExists(filename string) bool {
	fi, err := os.Lstat(filename)
	if fi != nil || (err != nil && !os.IsNotExist(err)) {
		return true
	}

	return false
}
