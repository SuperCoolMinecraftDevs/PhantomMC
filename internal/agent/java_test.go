package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeJVMRoot(t *testing.T, names ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range names {
		bin := filepath.Join(root, name, "bin")
		if err := os.MkdirAll(bin, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(bin, "java"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestFindRuntimesSortsNewestFirst(t *testing.T) {
	root := fakeJVMRoot(t,
		"java-21-openjdk-amd64",
		"java-25-openjdk-amd64",
		"java-17-openjdk-amd64",
	)

	found, err := FindRuntimes(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 3 {
		t.Fatalf("expected 3 runtimes, got %d", len(found))
	}
	if found[0].Major != 25 || found[2].Major != 17 {
		t.Errorf("not sorted newest first: %v", found)
	}
}

func TestFindRuntimesIgnoresJunk(t *testing.T) {
	root := fakeJVMRoot(t, "java-21-openjdk-amd64")
	for _, junk := range []string{"openjdk-21", "java-broken", "default-java"} {
		if err := os.MkdirAll(filepath.Join(root, junk), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	found, err := FindRuntimes(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Major != 21 {
		t.Errorf("expected only java 21, got %v", found)
	}
}

func TestFindRuntimesSkipsDirsWithoutBinary(t *testing.T) {
	root := fakeJVMRoot(t, "java-21-openjdk-amd64")
	if err := os.MkdirAll(filepath.Join(root, "java-25-openjdk-amd64"), 0o755); err != nil {
		t.Fatal(err)
	}

	found, _ := FindRuntimes(root)
	if len(found) != 1 || found[0].Major != 21 {
		t.Errorf("a directory with no java binary should be ignored, got %v", found)
	}
}

func TestSelectRuntimeRequiresExactMajor(t *testing.T) {
	runtimes := []Runtime{{Major: 25, Path: "/a"}, {Major: 21, Path: "/b"}}

	rt, err := SelectRuntime(runtimes, 21)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Path != "/b" {
		t.Errorf("expected java 21, got %s", rt.Path)
	}

	_, err = SelectRuntime(runtimes, 17)
	if err == nil {
		t.Fatal("expected an error when the required major is absent")
	}
	if !strings.Contains(err.Error(), "25, 21") {
		t.Errorf("error should list what is available: %v", err)
	}
}

func TestSelectRuntimeWithNothingInstalled(t *testing.T) {
	_, err := SelectRuntime(nil, 21)
	if err == nil || !strings.Contains(err.Error(), "no runtime is installed") {
		t.Fatalf("expected a clear message, got %v", err)
	}
}

func TestFindRuntimesOnMissingRoot(t *testing.T) {
	if _, err := FindRuntimes("/nonexistent-jvm-root"); err == nil {
		t.Fatal("expected an error for a missing root")
	}
}
