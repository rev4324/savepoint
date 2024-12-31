package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/rev4324/savepoint/validator"
)

const (
	USER_CONFIG_DIR_NAME  = "savepoint"
	USER_CONFIG_FILE_NAME = "config.json"
)

var cached *Config

type Config struct {
	Games  []OSSpecificGameConfig `json:"games"`
	Bucket S3BucketConfig         `json:"bucket"`
}

type OSSpecificGameConfig struct {
	Name    string `json:"name" validate:"required"`
	Os      string `json:"os" validate:"required"`
	SaveDir string `json:"saveDir" validate:"required"`
}

type S3BucketConfig struct {
	Endpoint  string `json:"endpoint" validate:"required"`
	AccessKey string `json:"accessKey" validate:"required"`
	SecretKey string `json:"secretKey" validate:"required"`
	Bucket    string `json:"bucket" validate:"required"`
}

func ValidateConfig(config *Config) error {
	validate := validator.Get()
	return validate.Struct(config)
}

func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()

	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, USER_CONFIG_DIR_NAME, USER_CONFIG_FILE_NAME), nil
}

func Parse() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	log.Debugf("Config path is at %s", path)

	fileReader, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// decode the JSON
	decoder := json.NewDecoder(fileReader)
	decoder.DisallowUnknownFields()
	config := &Config{}
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	// validate it
	validate := validator.Get()
	err = validate.Struct(config)
	if err != nil {
		return nil, err
	}

	// ...and process paths - expand env vars
	err = preprocessPaths(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// get cached config, parse the config file if it's not cached yet
func Get() (*Config, error) {
	if cached == nil {
		new, err := Parse()

		if err != nil {
			return nil, err
		}

		cached = new
	}

	return cached, nil
}

func preprocessPaths(config *Config) error {
	for i := 0; i < len(config.Games); i++ {
		expanded := os.ExpandEnv(config.Games[i].SaveDir)
		config.Games[i].SaveDir = expanded
	}

	return nil
}
