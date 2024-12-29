package configManager

import "github.com/rev4324/savepoint/validator"

type Config struct {
	Games  []OSSpecificGameConfig `json:"games"`
	Bucket S3BucketConfig         `json:"bucket"`
}

type OSSpecificGameConfig struct {
	Name       string `json:"name" validate:"required"`
	Os         string `json:"os" validate:"required"`
	SaveDir    string `json:"saveDir" validate:"required"`
}

type S3BucketConfig struct {
	Endpoint  string `json:"endpoint" validate:"required"`
	AccessKey string `json:"accessKey" validate:"required"`
	SecretKey string `json:"secretKey" validate:"required"`
	Bucket    string `json:"bucket" validate:"required"`
}

type ConfigManager interface {
	ConfigPath() (string, error)
	Get() (*Config, error)
}

func ValidateConfig(config *Config) error {
	validate := validator.Get()
	return validate.Struct(config)
}
