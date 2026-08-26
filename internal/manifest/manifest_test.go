package manifest

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

const digest = "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce" +
	"47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e"

func valid() *Manifest {
	return &Manifest{
		SchemaVersion: SchemaVersion,
		ID:            "atm-nine",
		Name:          "All The Mods 9",
		CreatedAt:     now,
		Minecraft:     Minecraft{Version: "1.21.8"},
		Loader:        Loader{Kind: LoaderFabric, Version: "0.16.14"},
		Java:          Java{Major: 21, Distribution: "temurin"},
		Auth:          Auth{Mode: AuthMicrosoft, ClientID: "app-id"},
		Graphics:      Graphics{Vendor: GPUAuto},
		JVM:           JVM{HeapMB: 4096},
		Mods: []Mod{{
			Source:    SourceModrinth,
			ProjectID: "AANobbMI",
			VersionID: "abc123",
			Filename:  "sodium.jar",
			Artifact: Artifact{
				URL:    "https://cdn.modrinth.com/data/AANobbMI/sodium.jar",
				Size:   1024,
				SHA512: digest,
			},
		}},
		Servers: []Server{{Name: "Home", Address: "mc.example.com:25565"}},
	}
}

func assertValid(t *testing.T, m *Manifest) {
	t.Helper()
	if err := m.Validate(now); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func assertRejects(t *testing.T, m *Manifest, field string) {
	t.Helper()
	err := m.Validate(now)
	if err == nil {
		t.Fatalf("expected rejection on %s", field)
	}
	if !strings.Contains(err.Error(), field) {
		t.Fatalf("expected error mentioning %s, got %v", field, err)
	}
}

func TestValidManifest(t *testing.T) {
	assertValid(t, valid())
}

func TestRoundTrip(t *testing.T) {
	encoded, err := json.Marshal(valid())
	if err != nil {
		t.Fatal(err)
	}
	var decoded Manifest
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	assertValid(t, &decoded)
}

func TestRejectsBadFields(t *testing.T) {
	cases := []struct {
		name  string
		field string
		mutit func(*Manifest)
	}{
		{"old schema", "schemaVersion", func(m *Manifest) { m.SchemaVersion = 0 }},
		{"bad id", "id", func(m *Manifest) { m.ID = "Not Valid" }},
		{"no version", "minecraft.version", func(m *Manifest) { m.Minecraft.Version = "" }},
		{"loader without version", "loader.version", func(m *Manifest) { m.Loader.Version = "" }},
		{"unknown loader", "loader.kind", func(m *Manifest) { m.Loader.Kind = "modloader" }},
		{"old java", "java.major", func(m *Manifest) { m.Java.Major = 6 }},
		{"tiny heap", "jvm.heapMB", func(m *Manifest) { m.JVM.HeapMB = 64 }},
		{"unknown gpu", "graphics.vendor", func(m *Manifest) { m.Graphics.Vendor = "matrox" }},
		{"path in filename", "filename", func(m *Manifest) { m.Mods[0].Filename = "../evil.jar" }},
		{"not a jar", "filename", func(m *Manifest) { m.Mods[0].Filename = "sodium.zip" }},
		{"short digest", "sha512", func(m *Manifest) { m.Mods[0].Artifact.SHA512 = "abc" }},
		{"plaintext url", "url", func(m *Manifest) { m.Mods[0].Artifact.URL = "http://cdn.example.com/a.jar" }},
		{"zero size", "size", func(m *Manifest) { m.Mods[0].Artifact.Size = 0 }},
		{"blank server", "servers[0].address", func(m *Manifest) { m.Servers[0].Address = "" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := valid()
			tc.mutit(m)
			assertRejects(t, m, tc.field)
		})
	}
}

func TestVanillaRejectsMods(t *testing.T) {
	m := valid()
	m.Loader = Loader{Kind: LoaderVanilla}
	assertRejects(t, m, "mods")
}

func TestDuplicateModFilenames(t *testing.T) {
	m := valid()
	m.Mods = append(m.Mods, m.Mods[0])
	assertRejects(t, m, "duplicate")
}

func TestManualModHasNoURL(t *testing.T) {
	m := valid()
	m.Mods[0].Manual = true
	assertRejects(t, m, "must be empty when manual")

	m.Mods[0].Artifact.URL = ""
	assertValid(t, m)
}

func TestMicrosoftModeRequiresAClientID(t *testing.T) {
	m := valid()
	m.Auth.ClientID = ""
	assertRejects(t, m, "auth.clientId")
}

func TestOfflineModeRejectsAClientID(t *testing.T) {
	m := valid()
	m.Auth = Auth{
		Mode:            AuthOffline,
		OfflineUsername: "Steve",
		ClientID:        "app-id",
		Entitlement:     &Entitlement{ExpiresAt: now.Add(time.Hour), Signature: "sig"},
	}
	assertRejects(t, m, "auth.clientId")
}

func TestOfflineRequiresEntitlement(t *testing.T) {
	m := valid()
	m.Auth = Auth{Mode: AuthOffline, OfflineUsername: "Steve"}
	assertRejects(t, m, "auth.entitlement")

	m.Auth.Entitlement = &Entitlement{
		IssuedAt:  now.Add(-time.Hour),
		ExpiresAt: now.Add(-time.Minute),
		Signature: "sig",
	}
	assertRejects(t, m, "auth.entitlement")

	m.Auth.Entitlement.ExpiresAt = now.Add(24 * time.Hour)
	assertValid(t, m)
}

func TestOfflineRejectsBadUsername(t *testing.T) {
	m := valid()
	m.Auth = Auth{
		Mode:            AuthOffline,
		OfflineUsername: "no spaces",
		Entitlement:     &Entitlement{ExpiresAt: now.Add(time.Hour), Signature: "sig"},
	}
	assertRejects(t, m, "auth.offlineUsername")
}

func TestRequiresNetwork(t *testing.T) {
	m := valid()
	if !m.RequiresNetwork() {
		t.Fatal("microsoft auth with mods should require network")
	}

	m.Loader = Loader{Kind: LoaderVanilla}
	m.Mods = nil
	m.Auth = Auth{
		Mode:            AuthOffline,
		OfflineUsername: "Steve",
		Entitlement:     &Entitlement{ExpiresAt: now.Add(time.Hour), Signature: "sig"},
	}
	if m.RequiresNetwork() {
		t.Fatal("offline vanilla should not require network")
	}
}
