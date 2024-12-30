package providers

import (
	"context"

	config "github.com/rev4324/savepoint/config"
)

type Provider interface {
	Upload(context.Context, *config.OSSpecificGameConfig) error
	Download(context.Context, *config.OSSpecificGameConfig) error
}
