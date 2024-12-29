// config defined per OS user. Default and cached after unmarshall.

package configManager

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	USER_CONFIG_DIR_NAME  = "savepoint"
	USER_CONFIG_FILE_NAME = "config.json"
)

var cached *Config

func New() ConfigManager {
	return &UserConfigManager{}
}

type UserConfigManager struct{}

func (c *UserConfigManager) ConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()

	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, USER_CONFIG_DIR_NAME, USER_CONFIG_FILE_NAME), nil
}

func (c *UserConfigManager) New() (*Config, error) {
	path, err := c.ConfigPath()

	if err != nil {
		return nil, err
	}

	fileReader, err := os.Open(path)

	decoder := json.NewDecoder(fileReader)
	decoder.DisallowUnknownFields()

	config := &Config{}
	err = decoder.Decode(config)

	if err != nil {
		return nil, err
	}

	return config, nil
}

func (c *UserConfigManager) Get() (*Config, error) {
	if cached == nil {
		new, err := c.New()

		if err != nil {
			return nil, err
		}

		cached = new
	}

	return cached, nil
}
