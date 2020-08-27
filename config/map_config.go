package config

import (
	"os"

	"github.com/FleekHQ/space-daemon/core/env"
)

type mapConfig struct {
	configStr  map[string]string
	configInt  map[string]int
	configBool map[string]bool
}

func NewMap(envVal env.SpaceEnv, flags *Flags) Config {
	configStr := make(map[string]string)
	configInt := make(map[string]int)
	configBool := make(map[string]bool)

	// default values
	configStr[SpaceStorePath] = "~/.fleek-space"
	configStr[MountFuseDrive] = "false"
	configStr[FuseDriveName] = "Space"
	configInt[SpaceServerPort] = 9999
	configInt[SpaceProxyServerPort] = 9998
	configInt[SpaceRestProxyServerPort] = 9997
	if flags.DevMode {
		configStr[Ipfsaddr] = os.Getenv(env.IpfsAddr)
		configStr[Ipfsnodeaddr] = os.Getenv(env.IpfsNodeAddr)
		configStr[Ipfsnodepath] = os.Getenv(env.IpfsNodePath)
		configStr[Mongousr] = os.Getenv(env.MongoUsr)
		configStr[Mongopw] = os.Getenv(env.MongoPw)
		configStr[Mongohost] = os.Getenv(env.MongoHost)
		configStr[Mongorepset] = os.Getenv(env.MongoRepSet)
		configStr[SpaceServicesAPIURL] = os.Getenv(env.ServicesAPIURL)
		configStr[SpaceVaultAPIURL] = os.Getenv(env.VaultAPIURL)
		configStr[SpaceVaultSaltSecret] = os.Getenv(env.VaultSaltSecret)
		configStr[SpaceServicesHubAuthURL] = os.Getenv(env.ServicesHubAuthURL)
		configStr[TextileHubTarget] = os.Getenv(env.TextileHubTarget)
		configStr[TextileHubMa] = os.Getenv(env.TextileHubMa)
		configStr[TextileThreadsTarget] = os.Getenv(env.TextileThreadsTarget)
		configStr[TextileUserKey] = os.Getenv(env.TextileUserKey)
		configStr[TextileUserSecret] = os.Getenv(env.TextileUserSecret)

		if os.Getenv(env.IpfsNode) != "" {
			configBool[Ipfsnode] = true
		}
	} else {
		configStr[Ipfsaddr] = flags.Ipfsaddr
		configStr[Ipfsnodeaddr] = flags.Ipfsnodeaddr
		configStr[Ipfsnodeaddr] = flags.Ipfsnodepath
		configStr[Mongousr] = flags.Mongousr
		configStr[Mongopw] = flags.Mongopw
		configStr[Mongohost] = flags.Mongohost
		configStr[Mongorepset] = flags.Mongorepset
		configStr[SpaceServicesAPIURL] = flags.ServicesAPIURL
		configStr[SpaceVaultAPIURL] = flags.VaultAPIURL
		configStr[SpaceVaultSaltSecret] = flags.VaultSaltSecret
		configStr[SpaceServicesHubAuthURL] = flags.ServicesHubAuthURL
		configStr[TextileHubTarget] = flags.TextileHubTarget
		configStr[TextileHubMa] = flags.TextileHubMa
		configStr[TextileThreadsTarget] = flags.TextileThreadsTarget
		configStr[TextileUserKey] = flags.TextileUserKey
		configStr[TextileUserSecret] = flags.TextileUserSecret

		configBool[Ipfsnode] = flags.Ipfsnode
	}

	c := mapConfig{
		configStr:  configStr,
		configInt:  configInt,
		configBool: configBool,
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

func (m mapConfig) GetBool(key string, defaultValue interface{}) bool {
	if val, exists := m.configBool[key]; exists {
		return val
	}

	if boolVal, ok := defaultValue.(bool); ok {
		return boolVal
	}

	return false
}
