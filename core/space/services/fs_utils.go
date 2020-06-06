package services

import (
	"github.com/FleekHQ/space-poc/log"
	"os"
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
