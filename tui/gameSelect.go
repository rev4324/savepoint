package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/rev4324/savepoint/config"
)

type FormData struct {
	Action    string
	GameIndex int
}

func StartTUI(games []config.OSSpecificGameConfig) (*FormData, error) {
	var formData FormData
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose action").
				Options(
					huh.NewOption("Upload a game save", "upload"),
					huh.NewOption("Download a game save", "download"),
				).
				Value(&formData.Action),
		),
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Choose the game").
				OptionsFunc(func() []huh.Option[int] {
					options := []huh.Option[int]{}

					for i, game := range games {
						options = append(options, huh.NewOption(game.Name, i))
					}

					return options
				}, nil).
				Value(&formData.GameIndex),
		),
	)

	err := form.Run()
	if err != nil {
		return nil, err
	}

	return &formData, nil
}
