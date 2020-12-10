package config

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/FleekHQ/space-daemon/core/env"
)

type mapConfig struct {
	configStr  map[string]string
	configInt  map[string]int
	configBool map[string]bool
}

func NewMap(flags *Flags) Config {
	configStr := make(map[string]string)
	configInt := make(map[string]int)
	configBool := make(map[string]bool)

	usr, _ := user.Current()

	// default values
	configStr[LogLevel] = flags.LogLevel
	configStr[SpaceStorePath] = filepath.Join(usr.HomeDir, ".fleek-space")
	configStr[MountFuseDrive] = "false"
	configStr[FuseDriveName] = "Space"
	configInt[SpaceServerPort] = 9999
	configInt[SpaceProxyServerPort] = 9998
	configInt[SpaceRestProxyServerPort] = 9997
	if flags.DevMode {
		configStr[Ipfsaddr] = os.Getenv(env.IpfsAddr)
		configStr[Ipfsnodeaddr] = os.Getenv(env.IpfsNodeAddr)
		configStr[Ipfsnodepath] = os.Getenv(env.IpfsNodePath)
		configStr[SpaceServicesAPIURL] = os.Getenv(env.ServicesAPIURL)
		configStr[SpaceVaultAPIURL] = os.Getenv(env.VaultAPIURL)
		configStr[SpaceVaultSaltSecret] = os.Getenv(env.VaultSaltSecret)
		configStr[SpaceServicesHubAuthURL] = os.Getenv(env.ServicesHubAuthURL)
		configStr[SpaceStorageSiteUrl] = os.Getenv(env.SpaceStorageSiteUrl)
		configStr[TextileHubTarget] = os.Getenv(env.TextileHubTarget)
		configStr[TextileHubMa] = os.Getenv(env.TextileHubMa)
		configStr[TextileThreadsTarget] = os.Getenv(env.TextileThreadsTarget)
		configStr[TextileHubGatewayUrl] = os.Getenv(env.TextileHubGatewayUrl)
		configStr[TextileUserKey] = os.Getenv(env.TextileUserKey)
		configStr[TextileUserSecret] = os.Getenv(env.TextileUserSecret)

		if os.Getenv(env.IpfsNode) != "false" {
			configBool[Ipfsnode] = true
		}
	} else {
		configStr[Ipfsaddr] = flags.Ipfsaddr
		configStr[Ipfsnodeaddr] = flags.Ipfsnodeaddr
		configStr[Ipfsnodepath] = flags.Ipfsnodepath
		configStr[SpaceServicesAPIURL] = flags.ServicesAPIURL
		configStr[SpaceVaultAPIURL] = flags.VaultAPIURL
		configStr[SpaceVaultSaltSecret] = flags.VaultSaltSecret
		configStr[SpaceServicesHubAuthURL] = flags.ServicesHubAuthURL
		if flags.SpaceStorageSiteUrl != "" {
			configStr[SpaceStorageSiteUrl] = flags.SpaceStorageSiteUrl
		}
		configStr[TextileHubTarget] = flags.TextileHubTarget
		configStr[TextileHubMa] = flags.TextileHubMa
		configStr[TextileThreadsTarget] = flags.TextileThreadsTarget
		configStr[TextileHubGatewayUrl] = flags.TextileHubGatewayUrl
		configStr[TextileUserKey] = flags.TextileUserKey
		configStr[TextileUserSecret] = flags.TextileUserSecret
		configBool[Ipfsnode] = flags.Ipfsnode
		if flags.SpaceStorePath != "" {
			configStr[SpaceStorePath] = flags.SpaceStorePath
		}
		if flags.RpcServerPort != 0 {
			configInt[SpaceServerPort] = flags.RpcServerPort
		}
		if flags.RpcProxyServerPort != 0 {
			configInt[SpaceProxyServerPort] = flags.RpcProxyServerPort
		}
		if flags.RestProxyServerPort != 0 {
			configInt[SpaceRestProxyServerPort] = flags.RestProxyServerPort
		}
		if flags.BuckdPath != "" {
			configStr[BuckdPath] = flags.BuckdPath
		}
		if flags.BuckdApiMaAddr != "" {
			configStr[BuckdApiMaAddr] = flags.BuckdApiMaAddr
		}
		if flags.BuckdApiProxyMaAddr != "" {
			configStr[BuckdApiProxyMaAddr] = flags.BuckdApiProxyMaAddr
		}
		if flags.BuckdThreadsHostMaAddr != "" {
			configStr[BuckdThreadsHostMaAddr] = flags.BuckdThreadsHostMaAddr
		}
		if flags.BuckdGatewayPort != 0 {
			configInt[BuckdGatewayPort] = flags.BuckdGatewayPort
		}
	}

	// Temp fix until we move to viper
	if configStr[Ipfsaddr] == "" {
		configStr[Ipfsaddr] = "/ip4/127.0.0.1/tcp/5001"
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
