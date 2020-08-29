package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/FleekHQ/space-daemon/core/env"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/creamdog/gonfig"
)

// standardConfig implements Config
// It loads its config information from the space.json file
type jsonConfig struct {
	cfg gonfig.Gonfig
}

type defaultSpaceJson struct {
	TextileHubTarget     string `json:"textileHubTarget"`
	TextileThreadsTarget string `json:"textileThreadsTarget"`
	RPCPort              int    `json:"rpcPort"`
	StorePath            string `json:"storePath"`
}

type defaultJson struct {
	Space defaultSpaceJson `json:"space"`
}

// Deprecated for the default values config
func NewJson(env env.SpaceEnv) Config {
	wd := env.WorkingFolder()
	f, err := os.Open(wd + "/" + JsonConfigFileName)
	if err != nil {
		// TODO: this may turn into a fatal panic error
		log.Info("could not find space.json file in " + wd + ", using defaults")
	}

	defer f.Close()
	config, err := gonfig.FromJson(f)
	if err != nil {
		log.Info("could not read space.json file, using defaults")
	}

	c := jsonConfig{
		cfg: config,
	}

	return c
}

// Gets the configuration value given a path in the json config file
// defaults to empty value if non is found and just logs errors
func (c jsonConfig) GetString(key string, defaultValue interface{}) string {
	if c.cfg == nil {
		return ""
	}
	v, err := c.cfg.GetString(key, defaultValue)
	if err != nil {
		log.Error(fmt.Sprintf("error getting key %s from config", key), err)
		return ""
	}
	log.Debug("Getting conf " + key + ": " + v)

	return v
}

// Gets the configuration value given a path in the json config file
// defaults to empty value if non is found and just logs errors
func (c jsonConfig) GetInt(key string, defaultValue interface{}) int {
	if c.cfg == nil {
		return 0
	}
	v, err := c.cfg.GetInt(key, defaultValue)
	if err != nil {
		log.Error(fmt.Sprintf("error getting key %s from config", key), err)
		return 0
	}

	return v
}

// Gets the configuration value given a path in the json config file
// defaults to empty value if non is found and just logs errors
func (c jsonConfig) GetBool(key string, defaultValue interface{}) bool {
	if c.cfg == nil {
		return false
	}
	v, err := c.cfg.GetBool(key, defaultValue)
	if err != nil {
		log.Error(fmt.Sprintf("error getting key %s from config", key), err)
		return false
	}

	return v
}

func CreateConfigJson() error {
	fmt.Println("Generating default config file")
	spaceJson := defaultSpaceJson{
		TextileHubTarget:     "textile-hub-dev.fleek.co:3006",
		TextileThreadsTarget: "textile-hub-dev.fleek.co:3006",
		RPCPort:              9999,
		StorePath:            "~/.fleek-space",
	}

	finalJson := defaultJson{
		Space: spaceJson,
	}

	currExecutablePath, err := os.Executable()
	if err != nil {
		return err
	}

	pathSegments := strings.Split(currExecutablePath, "/")
	wd := strings.Join(pathSegments[:len(pathSegments)-1], "/")

	jsonPath := wd + "/" + JsonConfigFileName
	marshalled, err := json.MarshalIndent(finalJson, "", "  ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(jsonPath, marshalled, 0644)
	if err != nil {
		return err
	}

	fmt.Println("Default config file generated")

	return nil
}
