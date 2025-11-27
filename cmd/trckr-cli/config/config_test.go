package config

import (
	"errors"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func TestGetConfigWithOverrideNoLocal(t *testing.T) {
	globalCfg := Config{Server: "127.0.0.1"}
	globalData, err := toml.Marshal(&globalCfg)
	if err != nil {
		t.Fatalf("unexpected error while marshaling config: %v", err)
	}

	var localData []byte = nil
	resCfg, err := getConfigWithOverride(globalData, localData)
	if err != nil {
		t.Fatalf("unexpected error while attempting to override global config with nil data: %v", err)
	}

	if resCfg != globalCfg {
		t.Fatalf("unexpected value returned: should be %v but is %v", globalCfg, resCfg)
	}
}

func TestGetConfigWithOverrideNoGlobal(t *testing.T) {
	var globalData []byte = nil

	localCfg := Config{Server: "127.0.0.1"}
	localData, err := toml.Marshal(localCfg)
	if err != nil {
		t.Fatalf("unexpected error while marshaling config: %v", err)
	}

	resCfg, err := getConfigWithOverride(globalData, localData)
	if err != nil {
		t.Fatalf("unexpected error while trying to get overriden config with no global config: %v", err)
	}

	if resCfg != localCfg {
		t.Fatalf("unexpected value returned: should be %v but is %v", localCfg, resCfg)
	}
}

func TestGetConfigWithOverrideNoGlobalNoLocal(t *testing.T) {
	var globalData []byte = nil
	var localData []byte = nil

	_, err := getConfigWithOverride(globalData, localData)
	if !errors.Is(err, ErrNoConfigFound) {
		if err == nil {
			t.Fatal("no existing config files didn't result in an error")
		}
		t.Fatalf("unexpected error while trying to get overriden config from empty data: %v", err)
	}
}

func TestGetConfigWithOverrideBothGlobalAndLocal(t *testing.T) {
	globalCfg := Config{Server: "127.0.0.1"}
	globalData, err := toml.Marshal(globalCfg)
	if err != nil {
		t.Fatalf("unexpected error while marshaling config: %v", err)
	}

	localCfg := Config{Server: "127.0.1.1"}
	localData, err := toml.Marshal(localCfg)
	if err != nil {
		t.Fatalf("unexpected error while marshaling config: %v", err)
	}

	resCfg, err := getConfigWithOverride(globalData, localData)
	if err != nil {
		t.Fatalf("unexpected erorr while overriding global config with local config: %v", err)
	}

	if resCfg != localCfg {
		t.Fatalf("unexpected value returned: should be %v but is %v", localCfg, resCfg)
	}
}

func TestGetConfigWithOverrideGlobalAndInvalidLocal(t *testing.T) {
	globalCfg := Config{Server: "127.0.0.1"}
	globalData, err := toml.Marshal(&globalCfg)
	if err != nil {
		t.Fatalf("unexpected error while marshaling config: %v", err)
	}

	localData := []byte("-")

	_, err = getConfigWithOverride(globalData, localData)
	if err == nil {
		t.Fatal("overriding with invalid config data allowed")
	}
}

func TestGetConfigWithOverrideLocalAndInvalidGlobal(t *testing.T) {
	globalData := []byte("-")

	localCfg := Config{Server: "127.0.0.1"}
	localData, err := toml.Marshal(&localCfg)
	if err != nil {
		t.Fatalf("unexpected error while marshaling config: %v", err)
	}

	_, err = getConfigWithOverride(globalData, localData)
	if err == nil {
		t.Fatal("overriding with invalid global config allowed")
	}
}
