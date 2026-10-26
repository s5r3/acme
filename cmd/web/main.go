// Command web runs the server-rendered HTML front-end.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"example.com/acme/internal/app"
)

func main() {
	cfgPath := flag.String("config", "configs/config.yaml", "path to the config file")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.RunWeb(ctx, *cfgPath); err != nil {
		fmt.Fprintln(os.Stderr, "web:", err)
		os.Exit(1)
	}
}
