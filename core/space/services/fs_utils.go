package services

import (
	"io"
	"os"

	"github.com/FleekHQ/space-daemon/log"
)

func PathExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}

	return false
}

func IsPathDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		log.Error("path error check isPathDir", err)
		return false
	}
	mode := fi.Mode()

	return mode.IsDir()
}

func RemoveDuplicates(elements []string) []string {
	// Use map to record duplicates as we find them.
	encountered := map[string]bool{}
	result := []string{}

	for v := range elements {
		if encountered[elements[v]] == true {
			// Do not add duplicate.
		} else {
			// Record this element as an encountered element.
			encountered[elements[v]] = true
			// Append to result slice.
			result = append(result, elements[v])
		}
	}
	// Return the new slice.
	return result
}

// Reader that also counts the amount of Bytes read from the wrappeed reader
type CountingReader struct {
	reader    io.Reader
	BytesRead int64
}

func NewCountingReader(reader io.Reader) *CountingReader {
	return &CountingReader{
		reader:    reader,
		BytesRead: 0,
	}
}

func (r *CountingReader) Read(b []byte) (int, error) {
	n, err := r.reader.Read(b)
	r.BytesRead += int64(n)
	return n, err
}
