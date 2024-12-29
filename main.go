package main

import (
	"context"
	"log"

	"github.com/rev4324/savepoint/configManager"
	"github.com/rev4324/savepoint/s3"
	"github.com/rev4324/savepoint/upload"
)

func main() {
	configMan := configManager.New()

	config, err := configMan.Get()

	if err != nil {
		panic(err)
	}

	err = configManager.ValidateConfig(config)

	if err != nil {
		panic(err)
	}

	log.Printf("Config loaded successfully with %d games.\n", len(config.Games))

	client, err := s3.Client(config)
	if err != nil {
		panic(err)
	}

	uploader := upload.New(client, config)

	err = uploader.Upload(context.Background(), &config.Games[0])

	if err != nil {
		panic(err)
	}
}
