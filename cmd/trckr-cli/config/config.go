package config

import (
	"errors"
	"os"
	"path"

	"github.com/kirsle/configdir"
	"github.com/pelletier/go-toml/v2"
)

const configFilename = "cli.toml"

var (
	ErrNoConfigFound = errors.New("no config file found")
)

type Config struct {
	Server string `toml:"server" comment:"server address"`
}

func FromBytes(data []byte) (Config, error) {
	var cfg Config
	err := toml.Unmarshal(data, &cfg)
	return cfg, err
}

func (cfg *Config) OverrideFrom(other Config) {
	if other.Server != "" {
		cfg.Server = other.Server
	}
}

func getConfigDataFromDir(dir string) ([]byte, error) {
	path := path.Join(dir, configFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

// getLocalConfigData returns the local config as bytes, a nil slice if it's not
// found, or an error if issues are encountered while trying to read it.
func getLocalConfigData() ([]byte, error) {
	dir := configdir.LocalConfig("trckr")
	return getConfigDataFromDir(dir)
}

// getGlobalConfigData returns the global config as bytes, a nil slice if none
// is found, or an error if issues are encountered while trying to read it.
func getGlobalConfigData() ([]byte, error) {
	dirs := configdir.SystemConfig("trckr")
	for _, dir := range dirs {
		data, err := getConfigDataFromDir(dir)
		if err != nil {
			return nil, err
		}
		if data != nil {
			return data, nil
		}
	}
	return nil, nil
}

func getConfigWithOverride(globalData []byte, localData []byte) (Config, error) {
	var cfg Config

	if len(globalData) == 0 && len(localData) == 0 {
		return cfg, ErrNoConfigFound
	}

	var err error
	cfg, err = FromBytes(globalData)
	if err != nil {
		return cfg, err
	}

	if localData != nil {
		localCfg, err := FromBytes(localData)
		if err != nil {
			return cfg, err
		}
		cfg.OverrideFrom(localCfg)
	}

	return cfg, nil
}

func Load() (Config, error) {
	var cfg Config

	globalData, err := getGlobalConfigData()
	if err != nil {
		return cfg, err
	}

	localData, err := getLocalConfigData()
	if err != nil {
		return cfg, err
	}

	return getConfigWithOverride(globalData, localData)
}

func LoadFromPath(path string) (Config, error) {
	var cfg Config

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, ErrNoConfigFound
		}
		return cfg, err
	}

	return FromBytes(data)
}
