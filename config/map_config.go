package config

import (
	"os"

	"github.com/FleekHQ/space-poc/core/env"
)

type mapConfig struct {
	configStr map[string]string
	configInt map[string]int
}

func NewMap(envVal env.SpaceEnv, flags *Flags) Config {
	configStr := make(map[string]string)
	configInt := make(map[string]int)

	// default values
	configStr[SpaceStorePath] = "~/.fleek-space"
	configStr[TextileHubTarget] = "textile-hub-dev.fleek.co:3006"
	configStr[TextileThreadsTarget] = "textile-hub-dev.fleek.co:3006"
	configStr[SpaceServicesAPIURL] = "https://td4uiovozc.execute-api.us-west-2.amazonaws.com/dev"   // TODO: Get a domain
	configStr[SpaceServicesHubAuthURL] = "wss://gqo1oqz055.execute-api.us-west-2.amazonaws.com/dev" // TODO: Get a domain
	configStr[MountFuseDrive] = "false"
	configStr[FuseDriveName] = "Space"
	configInt[SpaceServerPort] = 9999
	if flags.DevMode {
		configStr[Ipfsaddr] = os.Getenv(env.IpfsAddr)
		configStr[Mongousr] = os.Getenv(env.MongoUsr)
		configStr[Mongopw] = os.Getenv(env.MongoPw)
		configStr[Mongohost] = os.Getenv(env.MongoHost)
	} else {
		configStr[Ipfsaddr] = flags.Ipfsaddr
		configStr[Mongousr] = flags.Mongousr
		configStr[Mongopw] = flags.Mongopw
		configStr[Mongohost] = flags.Mongohost
	}

	c := mapConfig{
		configStr: configStr,
		configInt: configInt,
	}

	return c
}

func (m mapConfig) GetString(key string, defaultValue interface{}) string {
	if val, exists := m.configStr[key]; exists {
		return val
	}

	if stringValue, ok := defaultValue.(string); ok {
		return stringValue
	}

	return ""
}

func (m mapConfig) GetInt(key string, defaultValue interface{}) int {
	if val, exists := m.configInt[key]; exists {
		return val
	}

	if intVal, ok := defaultValue.(int); ok {
		return intVal
	}

	return 0
}
