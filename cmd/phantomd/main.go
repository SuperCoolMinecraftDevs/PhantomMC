// Command phantomd is the agent that runs inside a booted PhantomMC image. It
// resolves a build manifest, prepares the runtime environment in the overlay,
// and supervises the game process.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/SuperCoolMinecraftDevs/PhantomMC/internal/agent"
)

var version = "dev"

func main() {
	config := agent.Config{}

	flag.StringVar(&config.Source, "manifest", "", "path or https url of the build manifest")
	flag.StringVar(&config.Root, "root", "/var/lib/phantom", "directory for the runtime environment")
	flag.StringVar(&config.JVMRoot, "jvm-root", agent.DefaultJVMRoot, "directory holding installed java runtimes")
	flag.IntVar(&config.Workers, "workers", 8, "concurrent downloads")
	flag.BoolVar(&config.DryRun, "dry-run", false, "resolve and report the plan without changing anything")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	if config.Source == "" {
		fmt.Fprintln(os.Stderr, "phantomd: -manifest is required")
		flag.Usage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := agent.Run(ctx, config, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "phantomd: %v\n", err)
		os.Exit(1)
	}
}
