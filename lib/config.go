// 2019 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license
package lib

import (
	"errors"
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

type OutputConfig struct{}

type ServerConfig struct {
	DisplayName   string `yaml:"name"`
	Address       string
	UseTLS        bool `yaml:"tls"`
	TLSSkipVerify bool `yaml:"tls-skip-verify"`
}

type Config struct {
	Output  OutputConfig
	Servers map[string]ServerConfig
}

func LoadConfig(data string) (*Config, error) {
	var newConfig Config

	err := yaml.Unmarshal([]byte(data), &newConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal YAML: %s", err.Error())
	}

	// confirm info is correct
	for id, info := range newConfig.Servers {
		if id == "" {
			return nil, errors.New("Server IDs cannot be empty")
		}

		// set default display name
		if info.DisplayName == "" {
			info.DisplayName = id
			newConfig.Servers[id] = info
		}
	}

	return &newConfig, nil
}

func LoadConfigFromFile(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return LoadConfig(string(data))
}
