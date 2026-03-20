package main

import (
	"bug-report-service/internal/bootstrap"
	"context"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := bootstrap.NewApp()
	if err != nil {
		panic(err)
	}

	if err := app.Run(ctx); err != nil {
		panic(err)
	}

	time.Sleep(50 * time.Millisecond)
}
