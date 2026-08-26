package agent

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/SuperCoolMinecraftDevs/PhantomMC/internal/manifest"
	"github.com/SuperCoolMinecraftDevs/PhantomMC/internal/mojang"
)

// Layout is where each kind of file lives under the agent's root.
type Layout struct {
	Root      string
	Game      string
	Libraries string
	Assets    string
	Mods      string
}

func NewLayout(root string) Layout {
	game := filepath.Join(root, "minecraft")
	return Layout{
		Root:      root,
		Game:      game,
		Libraries: filepath.Join(root, "libraries"),
		Assets:    filepath.Join(root, "assets"),
		Mods:      filepath.Join(game, "mods"),
	}
}

type Installer struct {
	Client  *mojang.Client
	Layout  Layout
	Workers int
	Log     io.Writer
}

// Install downloads everything the version needs and returns a launch
// specification. Downloads are grouped so that failures surface against a named
// stage rather than as one undifferentiated pile.
func (in *Installer) Install(ctx context.Context, m *manifest.Manifest) (*mojang.LaunchSpec, *mojang.Version, error) {
	version, err := in.Client.Resolve(ctx, m.Minecraft.Version)
	if err != nil {
		return nil, nil, err
	}

	platform := mojang.LinuxAMD64()

	if err := in.fetchClientAndLibraries(ctx, version, platform); err != nil {
		return nil, nil, err
	}
	if err := in.fetchAssets(ctx, version); err != nil {
		return nil, nil, err
	}

	spec, err := version.Resolve(platform, in.Layout.Libraries)
	if err != nil {
		return nil, nil, err
	}
	return spec, version, nil
}

func (in *Installer) fetchClientAndLibraries(ctx context.Context, v *mojang.Version, p mojang.Platform) error {
	set := &mojang.FetchSet{}

	for _, artifact := range v.SelectLibraries(p) {
		set.Add(artifact, filepath.Join(in.Layout.Libraries, artifact.Path))
	}

	client, ok := v.Downloads["client"]
	if !ok {
		return fmt.Errorf("version %s has no client download", v.ID)
	}
	set.Add(client, filepath.Join(in.Layout.Libraries, mojang.ClientJarPath(v.ID, client)))

	in.logf("libraries: %d files, %s", set.Len(), humanBytes(set.TotalBytes()))
	return in.Client.FetchAll(ctx, set, in.workers(), in.progress("libraries"))
}

func (in *Installer) fetchAssets(ctx context.Context, v *mojang.Version) error {
	if v.AssetIndex.URL == "" {
		in.logf("assets: version declares no index, skipping")
		return nil
	}

	index, err := in.Client.AssetIndex(ctx, v.AssetIndex)
	if err != nil {
		return err
	}

	indexPath := filepath.Join(in.Layout.Assets, "indexes", v.AssetIndex.ID+".json")
	if err := in.Client.Fetch(ctx, mojang.Download{
		URL:  v.AssetIndex.URL,
		SHA1: v.AssetIndex.SHA1,
		Size: v.AssetIndex.Size,
	}, indexPath); err != nil {
		return fmt.Errorf("save asset index: %w", err)
	}

	set := &mojang.FetchSet{}
	for _, name := range index.SortedNames() {
		object := index.Objects[name]
		set.Add(object.Download(), filepath.Join(in.Layout.Assets, object.Path()))
	}

	in.logf("assets: %d objects, %s", set.Len(), humanBytes(set.TotalBytes()))
	return in.Client.FetchAll(ctx, set, in.workers(), in.progress("assets"))
}

func (in *Installer) workers() int {
	if in.Workers > 0 {
		return in.Workers
	}
	return 8
}

// progress reports at every five percent rather than every file, because five
// thousand lines of output on a console nobody can scroll is not progress.
func (in *Installer) progress(stage string) mojang.Progress {
	if in.Log == nil {
		return nil
	}
	var lastBucket int
	return func(done, total int, bytes int64) {
		bucket := done * 20 / total
		if bucket == lastBucket && done != total {
			return
		}
		lastBucket = bucket
		in.logf("%s: %d%% (%d/%d, %s)", stage, done*100/total, done, total, humanBytes(bytes))
	}
}

func (in *Installer) logf(format string, args ...any) {
	if in.Log == nil {
		return
	}
	fmt.Fprintf(in.Log, format+"\n", args...)
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for size := n / unit; size >= unit; size /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGT"[exp])
}
