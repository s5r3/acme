// Command migrate applies database migrations (goose) and seeds demo data.
//
//	migrate up | down | status | seed
//	migrate create <snake_case_name>
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
	dir := flag.String("dir", "migrations", "migrations directory (create only)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: migrate [-config file] up|down|status|seed")
		fmt.Fprintln(os.Stderr, "       migrate [-dir migrations] create <name>")
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	cmd := flag.Arg(0)
	if cmd == "create" {
		if flag.NArg() != 2 {
			flag.Usage()
			os.Exit(2)
		}
		file, err := app.CreateMigration(*dir, flag.Arg(1))
		if err != nil {
			fmt.Fprintln(os.Stderr, "migrate:", err)
			os.Exit(1)
		}
		fmt.Println("created", file)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.RunMigrate(ctx, *cfgPath, cmd); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}
