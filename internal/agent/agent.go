package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/SuperCoolMinecraftDevs/PhantomMC/internal/manifest"
	"github.com/SuperCoolMinecraftDevs/PhantomMC/internal/mojang"
)

type Config struct {
	Source  string
	Root    string
	JVMRoot string
	Workers int
	DryRun  bool
}

// Plan is everything the agent intends to do, derived from a manifest. It is
// built before anything is written so that a run can be inspected without side
// effects.
type Plan struct {
	Manifest     *manifest.Manifest
	GameDir      string
	ModsDir      string
	Downloads    []Download
	ManualMods   []string
	NeedsAuth    bool
	NeedsNetwork bool
}

type Download struct {
	URL         string
	Destination string
	Size        int64
	SHA512      string
}

func Run(ctx context.Context, config Config, out io.Writer) error {
	m, err := manifest.Load(ctx, &http.Client{Timeout: 30 * time.Second}, config.Source)
	if err != nil {
		return err
	}

	plan := BuildPlan(m, config.Root)
	writePlan(out, plan)

	if config.DryRun {
		return nil
	}

	session, err := Launch(ctx, m, config, out)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "session ended: %s after %s\n",
		session.Outcome, session.Runtime.Round(time.Second))
	if session.ShortLived() {
		return fmt.Errorf("the game exited after %s, which is too early to have reached the menu",
			session.Runtime.Round(time.Second))
	}
	return nil
}

// Launch installs whatever is missing and runs the game to completion.
//
// Sign in runs concurrently with the download. Both take time and neither
// depends on the other, and the user is going to spend that minute looking at
// their phone anyway. Doing them in sequence would also mean a misconfigured
// client id is only discovered after half a gigabyte has been fetched.
func Launch(ctx context.Context, m *manifest.Manifest, config Config, out io.Writer) (Session, error) {
	layout := NewLayout(config.Root)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type authResult struct {
		account Account
		err     error
	}
	auth := make(chan authResult, 1)
	go func() {
		account, err := resolveAccount(ctx, m, out)
		auth <- authResult{account, err}
	}()

	installer := &Installer{
		Client:  mojang.NewClient(&http.Client{Timeout: 5 * time.Minute}),
		Layout:  layout,
		Workers: config.Workers,
		Log:     out,
	}

	spec, version, err := installer.Install(ctx, m)
	if err != nil {
		return Session{}, err
	}

	// The version document is authoritative about the runtime it needs. The
	// manifest can be wrong, and a mismatch here fails in ways that are hard to
	// read out of a crash log.
	runtimes, err := FindRuntimes(config.JVMRoot)
	if err != nil {
		return Session{}, err
	}
	runtime, err := SelectRuntime(runtimes, version.JavaVersion.MajorVersion)
	if err != nil {
		return Session{}, err
	}
	fmt.Fprintf(out, "java        %d at %s\n", runtime.Major, runtime.Path)

	result := <-auth
	if result.err != nil {
		return Session{}, result.err
	}

	launcher := &Launcher{
		Java:   runtime.Path,
		Layout: layout,
		Stdout: out,
		Stderr: out,
	}

	args := launcher.Command(spec, m, result.account)
	fmt.Fprintf(out, "launching   %s as %s\n", spec.MainClass, result.account.Name)
	return launcher.Run(ctx, args)
}

func resolveAccount(ctx context.Context, m *manifest.Manifest, out io.Writer) (Account, error) {
	switch m.Auth.Mode {
	case manifest.AuthOffline:
		return OfflineAccount(m.Auth.OfflineUsername), nil
	case manifest.AuthMicrosoft:
		return SignIn(ctx, m.Auth, out)
	default:
		return Account{}, fmt.Errorf("unknown auth mode %q", m.Auth.Mode)
	}
}

func BuildPlan(m *manifest.Manifest, root string) *Plan {
	gameDir := filepath.Join(root, "minecraft")
	modsDir := filepath.Join(gameDir, "mods")

	plan := &Plan{
		Manifest:     m,
		GameDir:      gameDir,
		ModsDir:      modsDir,
		NeedsAuth:    m.Auth.Mode == manifest.AuthMicrosoft,
		NeedsNetwork: m.RequiresNetwork(),
	}

	for _, mod := range m.Mods {
		if mod.Manual {
			plan.ManualMods = append(plan.ManualMods, mod.Filename)
			continue
		}
		plan.Downloads = append(plan.Downloads, Download{
			URL:         mod.Artifact.URL,
			Destination: filepath.Join(modsDir, mod.Filename),
			Size:        mod.Artifact.Size,
			SHA512:      mod.Artifact.SHA512,
		})
	}
	return plan
}

func (p *Plan) TotalDownloadBytes() int64 {
	var total int64
	for _, d := range p.Downloads {
		total += d.Size
	}
	return total
}

func writePlan(out io.Writer, p *Plan) {
	m := p.Manifest
	fmt.Fprintf(out, "manifest    %s (%s)\n", m.Name, m.ID)
	fmt.Fprintf(out, "minecraft   %s\n", m.Minecraft.Version)
	fmt.Fprintf(out, "loader      %s %s\n", m.Loader.Kind, m.Loader.Version)
	fmt.Fprintf(out, "java        %d (%s)\n", m.Java.Major, m.Java.Distribution)
	fmt.Fprintf(out, "auth        %s\n", m.Auth.Mode)
	fmt.Fprintf(out, "heap        %d MiB\n", m.JVM.HeapMB)
	fmt.Fprintf(out, "game dir    %s\n", p.GameDir)
	fmt.Fprintf(out, "downloads   %d files, %.1f MiB\n",
		len(p.Downloads), float64(p.TotalDownloadBytes())/(1<<20))

	if len(p.ManualMods) > 0 {
		fmt.Fprintf(out, "manual      %d mods need to be supplied by hand:\n", len(p.ManualMods))
		for _, name := range p.ManualMods {
			fmt.Fprintf(out, "              %s\n", name)
		}
	}
	for _, s := range m.Servers {
		fmt.Fprintf(out, "server      %s (%s)\n", s.Name, s.Address)
	}
}
