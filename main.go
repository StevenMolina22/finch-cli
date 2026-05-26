package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"finch/internal/cli"
	"finch/internal/finch"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cmd := cli.NewRootCommand(func(ctx context.Context) (cli.Store, error) {
		return finch.OpenStoreFromEnv(ctx)
	}, time.Now)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
