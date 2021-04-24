package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/axpira/backend/api/rest"
	"github.com/axpira/backend/entity/config"
	"github.com/axpira/backend/infrastructure/repository/postgres"
	"github.com/rs/zerolog"
)

func main() {
	ctx := context.Background()
	l := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().
		Timestamp().
		Caller().
		Str("service", "backend").
		Logger()
	ctx = l.WithContext(ctx)
	err := config.InitConfig()
	fatalOnError(l, err, "error on init config")

	l.Info().Msgf("%v", config.Config)

	repo, err := postgres.NewExpenseRepository(ctx)
	fatalOnError(l, err, "error on create repository")
	restService, err := rest.New(ctx, repo)
	fatalOnError(l, err, "error on create rest service")

	restService.Start(ctx)

	l.Info().Str("service", "rest").Str("action", "started").Msg("waiting connection")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, os.Kill)
	<-stop

	restService.Stop(ctx)
	l.Info().
		Str("service", "rest").
		Str("action", "stopped").
		Msg("closing connection")

}

func fatalOnError(l zerolog.Logger, err error, msg string) {
	if err != nil {
		l.Fatal().Msgf("%s: %v", msg, err)
	}
}
