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
	var (
		config      agent.Config
		showVersion bool
		signInOnly  bool
	)

	flag.StringVar(&config.Source, "manifest", "", "path or https url of the build manifest")
	flag.StringVar(&config.Root, "root", "/var/lib/phantom", "directory for the runtime environment")
	flag.StringVar(&config.JVMRoot, "jvm-root", agent.DefaultJVMRoot, "directory holding installed java runtimes")
	flag.StringVar(&config.ClientID, "client-id", "",
		"azure application id for microsoft sign in, overrides "+agent.ClientIDEnv+" and the manifest")
	flag.IntVar(&config.Workers, "workers", 8, "concurrent downloads")
	flag.BoolVar(&config.DryRun, "dry-run", false, "resolve and report the plan without changing anything")
	flag.BoolVar(&signInOnly, "signin-test", false,
		"run only the microsoft sign in and report the result, then exit")
	flag.BoolVar(&showVersion, "version", false, "print the version and exit")

	flag.Usage = usage
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, config, signInOnly); err != nil {
		fmt.Fprintf(os.Stderr, "phantomd: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, config agent.Config, signInOnly bool) error {
	if signInOnly {
		clientID, source := agent.ResolveClientID(config.ClientID, "")
		if source == agent.ClientIDMissing {
			return fmt.Errorf("%s", agent.MissingClientIDHelp)
		}
		fmt.Printf("client id   from %s\n", source)
		return agent.SignInOnly(ctx, clientID, os.Stdout)
	}

	if config.Source == "" {
		flag.Usage()
		return fmt.Errorf("-manifest is required")
	}
	return agent.Run(ctx, config, os.Stdout)
}

func usage() {
	fmt.Fprintf(os.Stderr, `phantomd - the PhantomMC runtime agent

Usage:
  phantomd -manifest <path or url> [options]
  phantomd -signin-test -client-id <application-id>

Examples:
  Resolve a manifest and report what would happen, changing nothing:
    phantomd -manifest ./pack.json -dry-run

  Verify microsoft sign in works, without building an image or booting anything:
    phantomd -signin-test -client-id 00000000-0000-0000-0000-000000000000

  Run for real, overriding the client id the manifest carries:
    phantomd -manifest https://example.com/pack.json -client-id <id>

Options:
`)
	flag.PrintDefaults()
}
