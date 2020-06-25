package env

import (
	"github.com/joho/godotenv"
	syslog "log"
	"os"
	"strings"
)

type spaceEnv struct {
}

// Deprecated for default values
func New() SpaceEnv {
	err := godotenv.Load()
	if err != nil {
		syslog.Println("Error loading .env file. Using defaults")
	}

	return spaceEnv{}
}

func (s spaceEnv) CurrentFolder() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}

	pathSegments := strings.Split(path, "/")
	wd := strings.Join(pathSegments[:len(pathSegments)-1], "/")

	return wd, nil
}

func (s spaceEnv) WorkingFolder() string {
	var wd = os.Getenv(SpaceWorkingDir)
	// use default
	if wd == "" {
		cf, err := s.CurrentFolder()
		if err != nil {
			syslog.Fatal("unable to get working folder", err)
			panic(err)
		}
		wd = cf
	}

	return wd
}

func (s spaceEnv) LogLevel() string {
	var ll = os.Getenv(LogLevel)

	if ll == "" {
		return "Info"
	}

	return ll
}