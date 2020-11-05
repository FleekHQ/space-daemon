package util

import (
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
