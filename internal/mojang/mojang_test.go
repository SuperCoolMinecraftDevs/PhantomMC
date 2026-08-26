package mojang

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadVersion(t *testing.T) *Version {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "version-26.2.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v Version
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("real Mojang metadata failed to decode: %v", err)
	}
	return &v
}

func TestDecodesRealMetadata(t *testing.T) {
	v := loadVersion(t)

	if v.MainClass != "net.minecraft.client.main.Main" {
		t.Errorf("unexpected main class %q", v.MainClass)
	}
	if v.JavaVersion.MajorVersion != 25 {
		t.Errorf("expected java 25, got %d", v.JavaVersion.MajorVersion)
	}
	if len(v.Libraries) != 6 {
		t.Errorf("expected 6 libraries, got %d", len(v.Libraries))
	}
	if v.AssetIndex.ID == "" {
		t.Error("asset index id is empty")
	}
}

func TestRuleEvaluation(t *testing.T) {
	linux := LinuxAMD64()

	cases := []struct {
		name  string
		rules []Rule
		want  bool
	}{
		{"no rules allows", nil, true},
		{"allow linux", []Rule{{Action: ActionAllow, OS: &OSCondition{Name: "linux"}}}, true},
		{"allow osx only", []Rule{{Action: ActionAllow, OS: &OSCondition{Name: "osx"}}}, false},
		{"allow all then disallow linux", []Rule{
			{Action: ActionAllow},
			{Action: ActionDisallow, OS: &OSCondition{Name: "linux"}},
		}, false},
		{"allow all then disallow windows", []Rule{
			{Action: ActionAllow},
			{Action: ActionDisallow, OS: &OSCondition{Name: "windows"}},
		}, true},
		{"arch mismatch", []Rule{{Action: ActionAllow, OS: &OSCondition{Arch: "x86"}}}, false},
		{"arch match", []Rule{{Action: ActionAllow, OS: &OSCondition{Arch: "x86_64"}}}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := linux.Allows(tc.rules); got != tc.want {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestFeatureRules(t *testing.T) {
	p := LinuxAMD64()
	demo := []Rule{{Action: ActionAllow, Features: map[string]bool{"is_demo_user": true}}}

	if p.Allows(demo) {
		t.Error("demo argument should not apply without the feature set")
	}

	p.Features = map[string]bool{"is_demo_user": true}
	if !p.Allows(demo) {
		t.Error("demo argument should apply once the feature is set")
	}
}

func TestSelectLibrariesFiltersByPlatform(t *testing.T) {
	v := loadVersion(t)
	selected := v.SelectLibraries(LinuxAMD64())

	var names []string
	for _, lib := range v.Libraries {
		if LinuxAMD64().Allows(lib.Rules) {
			names = append(names, lib.Name)
		}
	}

	for _, name := range names {
		if strings.Contains(name, "objc") || strings.Contains(name, "natives-windows") {
			t.Errorf("platform specific library leaked through: %s", name)
		}
	}
	if len(selected) != len(names) {
		t.Fatalf("selection mismatch: %d artifacts for %d libraries", len(selected), len(names))
	}
	if len(selected) == 0 {
		t.Fatal("no libraries selected for linux")
	}
}

func TestArgumentUnmarshalBothShapes(t *testing.T) {
	var bare Argument
	if err := json.Unmarshal([]byte(`"--username"`), &bare); err != nil {
		t.Fatal(err)
	}
	if len(bare.Values) != 1 || bare.Values[0] != "--username" {
		t.Errorf("bare string not parsed: %#v", bare)
	}

	var multi Argument
	if err := json.Unmarshal([]byte(`{"rules":[{"action":"allow"}],"value":["-a","-b"]}`), &multi); err != nil {
		t.Fatal(err)
	}
	if len(multi.Values) != 2 || len(multi.Rules) != 1 {
		t.Errorf("object with array value not parsed: %#v", multi)
	}

	var single Argument
	if err := json.Unmarshal([]byte(`{"rules":[],"value":"-Xss1M"}`), &single); err != nil {
		t.Fatal(err)
	}
	if len(single.Values) != 1 || single.Values[0] != "-Xss1M" {
		t.Errorf("object with string value not parsed: %#v", single)
	}
}

func TestResolveBuildsClasspathAndSubstitutes(t *testing.T) {
	v := loadVersion(t)
	spec, err := v.Resolve(LinuxAMD64(), "/lib", Vars{
		PlayerName: "Steve",
		GameDir:    "/game",
		AssetsDir:  "/game/assets",
		UUID:       "abc",
		UserType:   "msa",
	})
	if err != nil {
		t.Fatal(err)
	}

	if spec.JavaMajor != 25 {
		t.Errorf("expected java 25, got %d", spec.JavaMajor)
	}
	if len(spec.Classpath) < 2 {
		t.Fatalf("classpath too short: %v", spec.Classpath)
	}
	for _, entry := range spec.Classpath {
		if !strings.HasPrefix(entry, "/lib/") {
			t.Errorf("classpath entry outside the library dir: %s", entry)
		}
	}

	joined := strings.Join(spec.GameArgs, " ")
	if !strings.Contains(joined, "Steve") {
		t.Errorf("player name not substituted: %s", joined)
	}
	if strings.Contains(joined, "${") {
		t.Errorf("unsubstituted placeholder remains: %s", joined)
	}

	for _, arg := range spec.JVMArgs {
		if strings.Contains(arg, "XstartOnFirstThread") {
			t.Error("macOS jvm argument leaked onto linux")
		}
		if strings.Contains(arg, "MojangTricksIntelDrivers") {
			t.Error("windows jvm argument leaked onto linux")
		}
	}
}

func TestResolveRejectsIncompleteVersion(t *testing.T) {
	if _, err := (&Version{ID: "x"}).Resolve(LinuxAMD64(), "/lib", Vars{}); err == nil {
		t.Fatal("expected an error for a version with no main class")
	}
	if _, err := (&Version{ID: "x", MainClass: "M"}).Resolve(LinuxAMD64(), "/lib", Vars{}); err == nil {
		t.Fatal("expected an error for a version with no client download")
	}
}

func TestManifestLookup(t *testing.T) {
	m := &VersionManifest{Versions: []VersionSummary{{ID: "26.2"}, {ID: "1.21.8"}}}
	if _, err := m.Find("1.21.8"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Find("nope"); err == nil {
		t.Fatal("expected an error for an unknown version")
	}
}
