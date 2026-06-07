package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/vasfvitor/nanci/internal/cli"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	os.Exit(cli.Execute(ctx))
}
