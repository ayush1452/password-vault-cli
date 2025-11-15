package main

import (
	"fmt"
	"os"

	"github.com/vault-cli/vault/internal/cli"
	"github.com/vault-cli/vault/internal/util"
)

func main() {
	// Handle panics gracefully
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: %v\n", r)
			os.Exit(util.ExitError)
		}
	}()

	if err := cli.Execute(); err != nil {
		util.HandleError(err, "")
	}
}
