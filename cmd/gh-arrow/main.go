// Command gh-arrow is the gh arrow GitHub CLI extension.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/flarebyte/gh-arrow/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := new(cli.Runner).Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
