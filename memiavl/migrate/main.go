package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/crypto-org-chain/cronos/memiavl/migrate/core"
	"github.com/spf13/cobra"
)

func rootCommand() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "iavl",
		Short: "benchmark cosmos/iavl",
	}
	cmd.AddCommand(core.SnapshotCommand())
	return cmd, nil
}

func main() {
	root, err := rootCommand()
	if err != nil {
		os.Exit(1)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	go func() {
		select {
		case <-signalChan:
			cancel()
		case <-ctx.Done():
		}
		<-signalChan
		os.Exit(2)
	}()

	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}
