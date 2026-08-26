package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/SuperCoolMinecraftDevs/PhantomMC/internal/manifest"
)

type Config struct {
	Source string
	Root   string
	DryRun bool
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
	return fmt.Errorf("execution is not implemented yet, re-run with -dry-run")
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
