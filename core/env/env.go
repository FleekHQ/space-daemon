package env

import (
	syslog "log"
	"os"
	"strings"
)

const (
	SpaceWorkingDir      = "SPACE_APP_DIR"
	LogLevel             = "LOG_LEVEL"
	IpfsAddr             = "IPFS_ADDR"
	MongoUsr             = "MONGO_USR"
	MongoPw              = "MONGO_PW"
	MongoHost            = "MONGO_HOST"
	MongoRepSet          = "MONGO_REPLICA_SET"
	ServicesAPIURL       = "SERVICES_API_URL"
	ServicesHubAuthURL   = "SERVICES_HUB_AUTH_URL"
	TextileHubTarget     = "TXL_HUB_TARGET"
	TextileThreadsTarget = "TXL_THREADS_TARGET"
)

type SpaceEnv interface {
	CurrentFolder() (string, error)
	WorkingFolder() string
	LogLevel() string
}

type defaultEnv struct {
}

func (d defaultEnv) CurrentFolder() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}

	pathSegments := strings.Split(path, "/")
	wd := strings.Join(pathSegments[:len(pathSegments)-1], "/")

	return wd, nil
}

func (d defaultEnv) WorkingFolder() string {
	cf, err := d.CurrentFolder()
	if err != nil {
		syslog.Fatal("unable to get working folder", err)
		panic(err)
	}
	return cf
}

func (d defaultEnv) LogLevel() string {
	return "Info"
}

// TODO: use this one after figuring textile keys
func NewDefault() SpaceEnv {
	return defaultEnv{}
}
