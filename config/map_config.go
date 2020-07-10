package config

import (
	"os"

	"github.com/FleekHQ/space-daemon/core/env"
)

type mapConfig struct {
	configStr map[string]string
	configInt map[string]int
}

func NewMap(flags *Flags) Config {
	configStr := make(map[string]string)
	configInt := make(map[string]int)

	// default values
	configStr[SpaceStorePath] = "~/.fleek-space"
	configStr[MountFuseDrive] = "false"
	configStr[FuseDriveName] = "Space"
	configInt[SpaceServerPort] = 9999
	configInt[SpaceProxyServerPort] = 9998
	configInt[SpaceRestProxyServerPort] = 9997
	if flags.DevMode {
		configStr[Ipfsaddr] = os.Getenv(env.IpfsAddr)
		configStr[Mongousr] = os.Getenv(env.MongoUsr)
		configStr[Mongopw] = os.Getenv(env.MongoPw)
		configStr[Mongohost] = os.Getenv(env.MongoHost)
		configStr[Mongorepset] = os.Getenv(env.MongoRepSet)
		configStr[SpaceServicesAPIURL] = os.Getenv(env.ServicesAPIURL)
		configStr[SpaceServicesHubAuthURL] = os.Getenv(env.ServicesHubAuthURL)
		configStr[TextileHubTarget] = os.Getenv(env.TextileHubTarget)
		configStr[TextileHubMa] = os.Getenv(env.TextileHubMa)
		configStr[TextileThreadsTarget] = os.Getenv(env.TextileThreadsTarget)
	} else {
		configStr[Ipfsaddr] = flags.Ipfsaddr
		configStr[Mongousr] = flags.Mongousr
		configStr[Mongopw] = flags.Mongopw
		configStr[Mongohost] = flags.Mongohost
		configStr[Mongorepset] = flags.Mongorepset
		configStr[SpaceServicesAPIURL] = flags.ServicesAPIURL
		configStr[SpaceServicesHubAuthURL] = flags.ServicesHubAuthURL
		configStr[TextileHubTarget] = flags.TextileHubTarget
		configStr[TextileHubMa] = flags.TextileHubMa
		configStr[TextileThreadsTarget] = flags.TextileThreadsTarget
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
