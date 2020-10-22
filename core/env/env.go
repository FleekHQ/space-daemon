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
	IpfsNode             = "IPFS_NODE"
	IpfsNodeAddr         = "IPFS_NODE_ADDR"
	IpfsNodePath         = "IPFS_NODE_PATH"
	MongoUsr             = "MONGO_USR"
	MongoPw              = "MONGO_PW"
	MongoHost            = "MONGO_HOST"
	MongoRepSet          = "MONGO_REPLICA_SET"
	ServicesAPIURL       = "SERVICES_API_URL"
	VaultAPIURL          = "VAULT_API_URL"
	VaultSaltSecret      = "VAULT_SALT_SECRET"
	ServicesHubAuthURL   = "SERVICES_HUB_AUTH_URL"
	SpaceStorageSiteUrl  = "SPACE_STORAGE_SITE_URL"
	TextileHubTarget     = "TXL_HUB_TARGET"
	TextileHubMa         = "TXL_HUB_MA"
	TextileThreadsTarget = "TXL_THREADS_TARGET"
	TextileHubGatewayUrl = "TXL_HUB_GATEWAY_URL"
	TextileUserKey       = "TXL_USER_KEY"
	TextileUserSecret    = "TXL_USER_SECRET"
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
