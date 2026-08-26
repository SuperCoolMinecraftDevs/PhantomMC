package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// DefaultJVMRoot is where Debian installs JREs.
const DefaultJVMRoot = "/usr/lib/jvm"

type Runtime struct {
	Major int
	Path  string
}

// FindRuntimes reports the JREs present on the system, newest first. Debian
// names these directories java-<major>-openjdk-<arch>, so the major version is
// readable without executing anything.
func FindRuntimes(root string) ([]Runtime, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", root, err)
	}

	var found []Runtime
	for _, entry := range entries {
		major, ok := majorFromDirName(entry.Name())
		if !ok {
			continue
		}
		binary := filepath.Join(root, entry.Name(), "bin", "java")
		if info, err := os.Stat(binary); err != nil || info.IsDir() {
			continue
		}
		found = append(found, Runtime{Major: major, Path: binary})
	}

	sort.Slice(found, func(i, j int) bool { return found[i].Major > found[j].Major })
	return found, nil
}

// SelectRuntime returns the runtime matching the required major exactly.
// Minecraft is sensitive to this in both directions: too old and it will not
// start, too new and mod loaders and reflection heavy mods break in ways that
// are hard to diagnose from a crash log.
func SelectRuntime(runtimes []Runtime, required int) (Runtime, error) {
	for _, rt := range runtimes {
		if rt.Major == required {
			return rt, nil
		}
	}

	available := make([]string, len(runtimes))
	for i, rt := range runtimes {
		available[i] = strconv.Itoa(rt.Major)
	}
	if len(available) == 0 {
		return Runtime{}, fmt.Errorf("java %d is required and no runtime is installed", required)
	}
	return Runtime{}, fmt.Errorf("java %d is required, only %s installed",
		required, strings.Join(available, ", "))
}

func majorFromDirName(name string) (int, bool) {
	if !strings.HasPrefix(name, "java-") {
		return 0, false
	}
	rest := strings.TrimPrefix(name, "java-")
	digits := rest
	if i := strings.IndexByte(rest, '-'); i >= 0 {
		digits = rest[:i]
	}
	major, err := strconv.Atoi(digits)
	if err != nil || major <= 0 {
		return 0, false
	}
	return major, true
}
