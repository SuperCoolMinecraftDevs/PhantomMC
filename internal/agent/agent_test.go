package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SuperCoolMinecraftDevs/PhantomMC/internal/manifest"
)

func loadFixture(t *testing.T) *manifest.Manifest {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "docs", "examples", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	m, err := manifest.Decode(f, time.Now())
	if err != nil {
		t.Fatalf("example manifest does not validate: %v", err)
	}
	return m
}

func TestExampleManifestIsValid(t *testing.T) {
	loadFixture(t)
}

func TestPlanSeparatesManualMods(t *testing.T) {
	plan := BuildPlan(loadFixture(t), "/var/lib/phantom")

	if len(plan.Downloads) != 2 {
		t.Fatalf("expected 2 downloads, got %d", len(plan.Downloads))
	}
	if len(plan.ManualMods) != 1 {
		t.Fatalf("expected 1 manual mod, got %d", len(plan.ManualMods))
	}
	if plan.ManualMods[0] != "jei-locked.jar" {
		t.Fatalf("unexpected manual mod %q", plan.ManualMods[0])
	}
	if got := plan.TotalDownloadBytes(); got != 3145728 {
		t.Fatalf("expected manual mod excluded from download size, got %d", got)
	}
}

func TestPlanPlacesModsUnderGameDir(t *testing.T) {
	plan := BuildPlan(loadFixture(t), "/tmp/phantom")

	want := filepath.Join("/tmp/phantom", "minecraft", "mods", "sodium-fabric-0.6.13.jar")
	if plan.Downloads[0].Destination != want {
		t.Fatalf("expected %s, got %s", want, plan.Downloads[0].Destination)
	}
}

func TestPlanFlags(t *testing.T) {
	plan := BuildPlan(loadFixture(t), "/tmp/phantom")
	if !plan.NeedsAuth {
		t.Error("microsoft mode should need auth")
	}
	if !plan.NeedsNetwork {
		t.Error("a manifest with mods should need network")
	}
}

func TestDecodeRejectsUnknownFields(t *testing.T) {
	_, err := manifest.Decode(strings.NewReader(`{"schemaVersion":1,"nope":true}`), time.Now())
	if err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected unknown field rejection, got %v", err)
	}
}
