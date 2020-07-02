package config

import (
	"errors"
)

const (
	JsonConfigFileName      = "space.json"
	SpaceServerPort         = "space/rpcPort"
	SpaceProxyServerPort    = "space/rpcProxyPort"
	SpaceStorePath          = "space/storePath"
	TextileHubTarget        = "space/textileHubTarget"
	TextileThreadsTarget    = "space/textileThreadsTarget"
	MountFuseDrive          = "space/mountFuseDrive"
	FuseMountPath           = "space/fuseMountPath"
	FuseDriveName           = "space/fuseDriveName"
	SpaceServicesAPIURL     = "space/servicesApiUrl"
	SpaceServicesHubAuthURL = "space/servicesHubAuthUrl"
	Ipfsaddr                = "space/ipfsAddr"
	Mongousr                = "space/mongoUsr"
	Mongopw                 = "space/mongoPw"
	Mongohost               = "space/mongoHost"
)

var (
	ErrConfigNotLoaded = errors.New("config file was not loaded correctly or it does not exist")
)

type Flags struct {
	Ipfsaddr  string
	Mongousr  string
	Mongopw   string
	Mongohost string
	DevMode   bool
}

// Config used to fetch config information
type Config interface {
	GetString(key string, defaultValue interface{}) string
	GetInt(key string, defaultValue interface{}) int
}
