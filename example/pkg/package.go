package pkg

import (
	"os"

	"example/pkg/cache"
	"example/pkg/config"
	"example/pkg/database"
	"example/pkg/queue"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

var BasePackage = do.Package(
	do.Lazy(config.NewConfig),
	do.Lazy(NewLogger),
	do.Lazy(database.NewDatabase),
	do.Lazy(cache.NewCache),
	do.Lazy(queue.NewQueue),
)

func NewLogger(i do.Injector) (*zerolog.Logger, error) {
	cfg := do.MustInvoke[*config.Config](i)
	level, err := zerolog.ParseLevel(cfg.Logger.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	logger := zerolog.New(os.Stdout).
		Level(level).
		With().
		Timestamp().
		Str("app", cfg.App.Name).
		Logger()
	return &logger, nil
}
