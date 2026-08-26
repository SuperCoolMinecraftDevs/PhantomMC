package mojang

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
)

const AssetBaseURL = "https://resources.download.minecraft.net"

type AssetObjects struct {
	Objects map[string]AssetObject `json:"objects"`
}

type AssetObject struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// Path is where the object lives on disk, relative to the assets directory.
// Objects are addressed by digest rather than by name, so the same file shared
// between versions is stored once.
func (o AssetObject) Path() string {
	return filepath.Join("objects", o.Hash[:2], o.Hash)
}

func (o AssetObject) URL() string {
	return fmt.Sprintf("%s/%s/%s", AssetBaseURL, o.Hash[:2], o.Hash)
}

func (o AssetObject) Download() Download {
	return Download{URL: o.URL(), SHA1: o.Hash, Size: o.Size}
}

func (c *Client) AssetIndex(ctx context.Context, index AssetIndex) (*AssetObjects, error) {
	var objects AssetObjects
	if err := c.getJSON(ctx, index.URL, &objects); err != nil {
		return nil, fmt.Errorf("fetch asset index %s: %w", index.ID, err)
	}
	return &objects, nil
}

// Progress reports completed work. Called from multiple goroutines, so
// implementations must be safe for concurrent use.
type Progress func(done, total int, bytes int64)

type FetchSet struct {
	Artifacts []Download
	Dests     []string
}

func (f *FetchSet) Add(artifact Download, dest string) {
	f.Artifacts = append(f.Artifacts, artifact)
	f.Dests = append(f.Dests, dest)
}

func (f *FetchSet) Len() int { return len(f.Artifacts) }

func (f *FetchSet) TotalBytes() int64 {
	var total int64
	for _, a := range f.Artifacts {
		total += a.Size
	}
	return total
}

// FetchAll downloads a set concurrently. An asset index is five thousand small
// files, so doing this sequentially would dominate the boot time entirely.
// The first error cancels the rest and is returned.
func (c *Client) FetchAll(ctx context.Context, set *FetchSet, workers int, progress Progress) error {
	if workers < 1 {
		workers = 1
	}
	total := set.Len()
	if total == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan int)
	var (
		wg        sync.WaitGroup
		done      atomic.Int64
		bytesDone atomic.Int64
		errOnce   sync.Once
		firstErr  error
	)

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				if err := c.Fetch(ctx, set.Artifacts[i], set.Dests[i]); err != nil {
					errOnce.Do(func() {
						firstErr = fmt.Errorf("%s: %w", set.Dests[i], err)
						cancel()
					})
					return
				}
				n := done.Add(1)
				b := bytesDone.Add(set.Artifacts[i].Size)
				if progress != nil {
					progress(int(n), total, b)
				}
			}
		}()
	}

	for i := range total {
		select {
		case jobs <- i:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return firstErr
		}
	}
	close(jobs)
	wg.Wait()
	return firstErr
}

// SortedNames returns the asset names in a stable order, which matters only for
// producing reproducible output in tests and logs.
func (a *AssetObjects) SortedNames() []string {
	names := make([]string, 0, len(a.Objects))
	for name := range a.Objects {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
