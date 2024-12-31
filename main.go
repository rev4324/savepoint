package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/rev4324/savepoint/config"
	"github.com/rev4324/savepoint/providers/s3"
	"github.com/rev4324/savepoint/tui"
)

func main() {
	_, isDebug := os.LookupEnv("SAVEPOINT_DEBUG")
	if isDebug {
		log.SetLevel(log.DebugLevel)
	}

	config, err := config.Get()
	if err != nil {
		panic(err)
	}

	log.Infof("Config loaded successfully with %d games.\n", len(config.Games))

	s3Provider, err := s3.New(config)
	if err != nil {
		panic(err)
	}

	formData, err := tui.StartTUI(config.Games)
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("Aborted.")
			return
		}

		panic(err)
	}

	switch formData.Action {
	case "upload":
		err = s3Provider.Upload(context.Background(), &config.Games[formData.GameIndex])
		if err != nil {
			panic(err)
		}
	case "download":
		err = s3Provider.Download(context.Background(), &config.Games[formData.GameIndex])
		if err != nil {
			panic(err)
		}
	}
}
