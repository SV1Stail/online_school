package main

import (
	"context"
	"os"
	"strings"

	"github.com/SV1Stail/online_school/internal/config"
	"github.com/SV1Stail/online_school/internal/logging"
	"github.com/SV1Stail/online_school/internal/platform/postgres"
	"github.com/SV1Stail/online_school/internal/user"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger := logging.New(cfg.LogLevel, cfg.LogPretty)
	login := strings.TrimSpace(os.Getenv("BOOTSTRAP_LOGIN"))
	password := os.Getenv("BOOTSTRAP_PASSWORD")
	fullName := strings.TrimSpace(os.Getenv("BOOTSTRAP_FULL_NAME"))
	timezone := strings.TrimSpace(os.Getenv("BOOTSTRAP_TIMEZONE"))
	if login == "" || password == "" || fullName == "" || timezone == "" {
		logger.Fatal().Msg("BOOTSTRAP_LOGIN, BOOTSTRAP_PASSWORD, BOOTSTRAP_FULL_NAME and BOOTSTRAP_TIMEZONE are required")
	}

	ctx := context.Background()
	database, err := postgres.Open(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("database initialization failed")
	}
	defer database.Close()

	service := user.NewService(database, database, database)
	created, err := service.Create(ctx, "", user.CreateInput{
		Login:    login,
		Password: password,
		FullName: fullName,
		Role:     user.RoleModerator,
		Timezone: timezone,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("moderator bootstrap failed")
	}
	logger.Info().Str("user_id", created.ID).Msg("initial moderator created")
}
