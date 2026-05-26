package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"finch/internal/cli"
	"finch/internal/finch"
)

func main() {
	cmd := cli.NewRootCommand(func(ctx context.Context) (cli.Store, error) {
		return finch.OpenStoreFromEnv(ctx)
	}, time.Now)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
