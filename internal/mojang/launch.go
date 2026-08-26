package mojang

import (
	"fmt"
	"path/filepath"
	"strings"
)

// LaunchSpec is everything needed to start the game, resolved for one platform.
type LaunchSpec struct {
	MainClass  string
	Classpath  []string
	GameArgs   []string
	JVMArgs    []string
	JavaMajor  int
	AssetIndex string
}

// Vars are the placeholder substitutions Mojang's argument templates expect.
type Vars struct {
	PlayerName      string
	VersionName     string
	GameDir         string
	AssetsDir       string
	AssetIndex      string
	UUID            string
	AccessToken     string
	UserType        string
	VersionType     string
	LauncherName    string
	LauncherVersion string
	NativesDir      string
	Classpath       string
}

// SelectLibraries returns the library artifacts that apply to this platform, in the
// order Mojang lists them, which is the order they must appear on the classpath.
func (v *Version) SelectLibraries(p Platform) []Download {
	var out []Download
	for _, lib := range v.Libraries {
		if !p.Allows(lib.Rules) {
			continue
		}
		if lib.Downloads.Artifact == nil {
			continue
		}
		out = append(out, *lib.Downloads.Artifact)
	}
	return out
}

// Resolve produces a launch specification for the platform, with every
// placeholder substituted.
func (v *Version) Resolve(p Platform, libraryDir string, vars Vars) (*LaunchSpec, error) {
	if v.MainClass == "" {
		return nil, fmt.Errorf("version %s has no main class", v.ID)
	}

	client, ok := v.Downloads["client"]
	if !ok {
		return nil, fmt.Errorf("version %s has no client download", v.ID)
	}

	var classpath []string
	for _, artifact := range v.SelectLibraries(p) {
		classpath = append(classpath, filepath.Join(libraryDir, artifact.Path))
	}
	classpath = append(classpath, filepath.Join(libraryDir, clientPath(v.ID, client)))

	vars.Classpath = strings.Join(classpath, string(filepath.ListSeparator))
	if vars.AssetIndex == "" {
		vars.AssetIndex = v.AssetIndex.ID
	}
	if vars.VersionName == "" {
		vars.VersionName = v.ID
	}
	if vars.VersionType == "" {
		vars.VersionType = v.Type
	}

	return &LaunchSpec{
		MainClass:  v.MainClass,
		Classpath:  classpath,
		JVMArgs:    expand(v.Arguments.JVM, p, vars),
		GameArgs:   expand(v.Arguments.Game, p, vars),
		JavaMajor:  v.JavaVersion.MajorVersion,
		AssetIndex: v.AssetIndex.ID,
	}, nil
}

func clientPath(id string, client Download) string {
	if client.Path != "" {
		return client.Path
	}
	return filepath.Join("com", "mojang", "minecraft", id, "minecraft-"+id+"-client.jar")
}

func expand(args []Argument, p Platform, vars Vars) []string {
	var out []string
	for _, arg := range args {
		if !p.Allows(arg.Rules) {
			continue
		}
		for _, value := range arg.Values {
			out = append(out, substitute(value, vars))
		}
	}
	return out
}

func substitute(s string, v Vars) string {
	return placeholders(v).Replace(s)
}

func placeholders(v Vars) *strings.Replacer {
	return strings.NewReplacer(
		"${auth_player_name}", v.PlayerName,
		"${version_name}", v.VersionName,
		"${game_directory}", v.GameDir,
		"${assets_root}", v.AssetsDir,
		"${assets_index_name}", v.AssetIndex,
		"${auth_uuid}", v.UUID,
		"${auth_access_token}", v.AccessToken,
		"${clientid}", "",
		"${auth_xuid}", "",
		"${user_type}", v.UserType,
		"${version_type}", v.VersionType,
		"${launcher_name}", v.LauncherName,
		"${launcher_version}", v.LauncherVersion,
		"${natives_directory}", v.NativesDir,
		"${classpath}", v.Classpath,
		"${classpath_separator}", string(filepath.ListSeparator),
		"${library_directory}", v.NativesDir,
	)
}
