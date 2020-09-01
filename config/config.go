package config

import (
	"errors"
)

const (
	JsonConfigFileName       = "space.json"
	SpaceServerPort          = "space/rpcPort"
	SpaceProxyServerPort     = "space/rpcProxyPort"
	SpaceRestProxyServerPort = "space/restProxyPort"
	SpaceStorePath           = "space/storePath"
	TextileHubTarget         = "space/textileHubTarget"
	TextileHubMa             = "space/textileHubMa"
	TextileThreadsTarget     = "space/textileThreadsTarget"
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
	Mongousr                 = "space/mongoUsr"
	Mongopw                  = "space/mongoPw"
	Mongohost                = "space/mongoHost"
	Mongorepset              = "space/mongoRepSet"
	MinThreadsConnection     = "space/minThreadsConn"
	MaxThreadsConnection     = "space/maxThreadsConn"
	StartPowergate           = "space/startPowergate"
	LotusAddress             = "space/lotusAddress"
	PowdGrpcHostAddress      = "space/powergateGrpcHostAddress"
	PowdGrpcWebProxyAddress  = "space/powergateGrpcWebProxyAddress"
	PowdGatewayHostAddress   = "space/powergateGrpcGatewayHostAddress"
)

var (
	ErrConfigNotLoaded = errors.New("config file was not loaded correctly or it does not exist")
)

type Flags struct {
	Ipfsaddr                string
	Ipfsnode                bool
	Ipfsnodeaddr            string
	Ipfsnodepath            string
	Mongousr                string
	Mongopw                 string
	Mongohost               string
	Mongorepset             string
	DevMode                 bool
	ServicesAPIURL          string
	VaultAPIURL             string
	VaultSaltSecret         string
	ServicesHubAuthURL      string
	TextileHubTarget        string
	TextileHubMa            string
	TextileThreadsTarget    string
	TextileUserKey          string
	TextileUserSecret       string
	StartPowergate          bool
	LotusAddress            string
	PowdGrpcHostAddress     string
	PowdGrpcWebProxyAddress string
	PowdGatewayHostAddress  string
}

// Config used to fetch config information
type Config interface {
	GetString(key string, defaultValue interface{}) string
	GetInt(key string, defaultValue interface{}) int
	GetBool(key string, defaultValue interface{}) bool
}
