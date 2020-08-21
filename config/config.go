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
	MountFuseDrive           = "space/mountFuseDrive"
	FuseMountPath            = "space/fuseMountPath"
	FuseDriveName            = "space/fuseDriveName"
	SpaceServicesAPIURL      = "space/servicesApiUrl"
	SpaceServicesHubAuthURL  = "space/servicesHubAuthUrl"
	Ipfsaddr                 = "space/ipfsAddr"
	Ipfsnodeaddr			 = "space/ipfsNodeAddr"
	Ipfsnodepath             = "space/ipfsNodePath"
	Mongousr                 = "space/mongoUsr"
	Mongopw                  = "space/mongoPw"
	Mongohost                = "space/mongoHost"
	Mongorepset              = "space/mongoRepSet"
	MinThreadsConnection     = "space/minThreadsConn"
	MaxThreadsConnection     = "space/maxThreadsConn"
)

var (
	ErrConfigNotLoaded = errors.New("config file was not loaded correctly or it does not exist")
)

type Flags struct {
	Ipfsaddr             string
	Ipfsnodeaddr         string
	Ipfsnodepath         string
	Mongousr             string
	Mongopw              string
	Mongohost            string
	Mongorepset          string
	DevMode              bool
	ServicesAPIURL       string
	ServicesHubAuthURL   string
	TextileHubTarget     string
	TextileHubMa         string
	TextileThreadsTarget string
}

// Config used to fetch config information
type Config interface {
	GetString(key string, defaultValue interface{}) string
	GetInt(key string, defaultValue interface{}) int
}
