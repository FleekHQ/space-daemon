package fsds

import (
	"fmt"
	"os"
	"strings"
)

func isBaseDirectory(path string) bool {
	return path == "/"
}

func isDirPath(path string) bool {
	return strings.HasSuffix(path, fmt.Sprintf("%c", os.PathSeparator))
}

func isNotExistError(err error) bool {
	if err == nil {
		return false
	}
	// Example of current error representing file not found:
	// error: code = Unknown desc = no link named ".localized" under bafybeievqvkeo2ycggt4lino45pj3olv7yo2e6sybcmyphicejsvq2vimi[]
	if strings.Contains(err.Error(), "no link named") {
		return true
	}

	if strings.Contains(err.Error(), "could not resolve path") {
		return true
	}

	return false
}
