package config

import (
	"errors"
)

const (
	JsonConfigFileName       = "space.json"
	SpaceServerPort          = "space/rpcPort"
	SpaceProxyServerPort     = "space/rpcProxyPort"
	SpaceRestProxyServerPort = "space/restProxyPort"
	SpaceStorageSiteUrl      = "space/storageSiteUrl"
	SpaceStorePath           = "space/storePath"
	TextileHubTarget         = "space/textileHubTarget"
	TextileHubMa             = "space/textileHubMa"
	TextileThreadsTarget     = "space/textileThreadsTarget"
	TextileHubGatewayUrl     = "space/TextileHubGatewayUrl"
	TextileUserKey           = "space/textileUserKey"
	TextileUserSecret        = "space/textileUserSecret"
	MountFuseDrive           = "space/mountFuseDrive"
	FuseMountPath            = "space/fuseMountPath"
	FuseDriveName            = "space/fuseDriveName"
	SpaceServicesAPIURL      = "space/servicesApiUrl"
	SpaceVaultAPIURL         = "space/vaultApiUrl"
	SpaceVaultSaltSecret     = "space/vaultSaltSecret"
	SpaceServicesHubAuthURL  = "space/servicesHubAuthUrl"
	Ipfsaddr                 = "space/ipfsAddr"
	Ipfsnode                 = "space/ipfsNode"
	Ipfsnodeaddr             = "space/ipfsNodeAddr"
	Ipfsnodepath             = "space/ipfsNodePath"
	MinThreadsConnection     = "space/minThreadsConn"
	MaxThreadsConnection     = "space/maxThreadsConn"
	BuckdPath                = "space/BuckdPath"
	BuckdApiMaAddr           = "space/BuckdApiMaAddr"
	BuckdApiProxyMaAddr      = "space/BuckdApiProxyMaAddr"
	BuckdThreadsHostMaAddr   = "Space/BuckdThreadsHostMaAddr"
	BuckdGatewayPort         = "Space/BuckdGatewayPort"
	LogLevel                 = "Space/LogLevel"
)

var (
	ErrConfigNotLoaded = errors.New("config file was not loaded correctly or it does not exist")
)

type Flags struct {
	Ipfsaddr               string
	Ipfsnode               bool
	Ipfsnodeaddr           string
	Ipfsnodepath           string
	DevMode                bool
	ServicesAPIURL         string
	SpaceStorageSiteUrl    string
	VaultAPIURL            string
	VaultSaltSecret        string
	ServicesHubAuthURL     string
	TextileHubTarget       string
	TextileHubMa           string
	TextileThreadsTarget   string
	TextileHubGatewayUrl   string
	TextileUserKey         string
	TextileUserSecret      string
	SpaceStorePath         string
	RpcServerPort          int
	RpcProxyServerPort     int
	RestProxyServerPort    int
	BuckdPath              string
	BuckdApiMaAddr         string
	BuckdApiProxyMaAddr    string
	BuckdThreadsHostMaAddr string
	BuckdGatewayPort       int
	LogLevel               string
}

// Config used to fetch config information
type Config interface {
	GetString(key string, defaultValue interface{}) string
	GetInt(key string, defaultValue interface{}) int
	GetBool(key string, defaultValue interface{}) bool
}
