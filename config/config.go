package config

import (
	"errors"
)

const (
	JsonConfigFileName   = "space.json"
	SpaceServerPort      = "space/rpcPort"
	SpaceStorePath       = "space/storePath"
	TextileHubTarget     = "space/textileHubTarget"
	TextileThreadsTarget = "space/textileThreadsTarget"
	MountFuseDrive       = "space/mountFuseDrive"
	FuseMountPath        = "space/fuseMountPath"
	FuseDriveName        = "space/fuseDriveName"
	SpaceServicesAPIURL  = "space/servicesApiUrl"
)

var (
	ErrConfigNotLoaded = errors.New("config file was not loaded correctly or it does not exist")
)

// Config used to fetch config information
type Config interface {
	GetString(key string, defaultValue interface{}) string
	GetInt(key string, defaultValue interface{}) int
}


