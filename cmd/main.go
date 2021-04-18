package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/axpira/backend/api/rest"
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

	db, _, err := sqlmock.New()
	fatalOnError(l, err, "error on database connect")

	// db, err := sql.Open("driver-name", "database=test1")
	// fatalOnError(err, "error on database connect")

	repo := postgres.NewExpenseRepository(db)
	restService, err := rest.New(ctx, repo)
	fatalOnError(l, err, "error on create rest service")

	restService.Start(ctx)

	l.Info().Str("service", "rest").Str("action", "started").Msg("waiting connection")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	// Waiting for SIGINT (pkill -2)
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
