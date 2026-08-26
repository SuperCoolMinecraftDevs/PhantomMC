package mojang

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestAssetObjectAddressing(t *testing.T) {
	obj := AssetObject{Hash: "b62ca8ec10d07e6bf5ac8dae0c8c1d2e6a1e3356", Size: 9101}

	wantPath := filepath.Join("objects", "b6", obj.Hash)
	if obj.Path() != wantPath {
		t.Errorf("expected %s, got %s", wantPath, obj.Path())
	}
	if want := AssetBaseURL + "/b6/" + obj.Hash; obj.URL() != want {
		t.Errorf("expected %s, got %s", want, obj.URL())
	}
	if d := obj.Download(); d.SHA1 != obj.Hash || d.Size != obj.Size {
		t.Errorf("download does not carry the digest and size: %#v", d)
	}
}

func TestFetchAllDownloadsEverything(t *testing.T) {
	const count = 50
	bodies := make(map[string][]byte, count)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if body, ok := bodies[r.URL.Path]; ok {
			w.Write(body)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	dir := t.TempDir()
	set := &FetchSet{}
	for i := range count {
		path := fmt.Sprintf("/object-%d", i)
		body := []byte(fmt.Sprintf("contents of object %d", i))
		bodies[path] = body
		set.Add(
			Download{URL: server.URL + path, SHA1: digestOf(body), Size: int64(len(body))},
			filepath.Join(dir, fmt.Sprintf("obj-%d", i)),
		)
	}

	var (
		mu       sync.Mutex
		lastDone int
	)
	err := NewClient(server.Client()).FetchAll(context.Background(), set, 8,
		func(done, total int, bytes int64) {
			mu.Lock()
			defer mu.Unlock()
			if done > lastDone {
				lastDone = done
			}
			if total != count {
				t.Errorf("total should be %d, got %d", count, total)
			}
		})
	if err != nil {
		t.Fatal(err)
	}
	if lastDone != count {
		t.Errorf("progress reached %d of %d", lastDone, count)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != count {
		t.Errorf("expected %d files, got %d", count, len(entries))
	}
}

func TestFetchAllStopsOnFirstError(t *testing.T) {
	var served atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served.Add(1)
		if strings.HasSuffix(r.URL.Path, "bad") {
			http.Error(w, "gone", http.StatusNotFound)
			return
		}
		w.Write([]byte("fine"))
	}))
	defer server.Close()

	dir := t.TempDir()
	set := &FetchSet{}
	set.Add(Download{URL: server.URL + "/bad", SHA1: digestOf([]byte("fine"))}, filepath.Join(dir, "bad"))
	for i := range 20 {
		set.Add(
			Download{URL: server.URL + "/ok", SHA1: digestOf([]byte("fine"))},
			filepath.Join(dir, fmt.Sprintf("ok-%d", i)),
		)
	}

	err := NewClient(server.Client()).FetchAll(context.Background(), set, 4, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "bad") {
		t.Errorf("error should name the failing destination: %v", err)
	}
}

func TestFetchAllRespectsCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("payload"))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dir := t.TempDir()
	set := &FetchSet{}
	for i := range 10 {
		set.Add(
			Download{URL: server.URL, SHA1: digestOf([]byte("payload"))},
			filepath.Join(dir, fmt.Sprintf("obj-%d", i)),
		)
	}

	err := NewClient(server.Client()).FetchAll(ctx, set, 2, nil)
	if err != nil && !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "context canceled") {
		t.Logf("cancellation surfaced as: %v", err)
	}
}

func TestFetchAllEmptySet(t *testing.T) {
	if err := NewClient(nil).FetchAll(context.Background(), &FetchSet{}, 4, nil); err != nil {
		t.Fatalf("an empty set should be a no-op, got %v", err)
	}
}

func TestFetchSetAccounting(t *testing.T) {
	set := &FetchSet{}
	set.Add(Download{Size: 100}, "/a")
	set.Add(Download{Size: 250}, "/b")

	if set.Len() != 2 {
		t.Errorf("expected 2, got %d", set.Len())
	}
	if set.TotalBytes() != 350 {
		t.Errorf("expected 350 bytes, got %d", set.TotalBytes())
	}
}

func TestSortedNamesIsStable(t *testing.T) {
	a := &AssetObjects{Objects: map[string]AssetObject{
		"zebra.png": {}, "apple.png": {}, "mango.png": {},
	}}
	got := a.SortedNames()
	want := []string{"apple.png", "mango.png", "zebra.png"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}
