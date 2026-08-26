// Package mojang models the metadata Mojang publishes for the game: the version
// manifest, individual version documents, and the rules that decide which
// libraries apply to the machine doing the launching.
package mojang

import (
	"encoding/json"
	"fmt"
)

type VersionManifest struct {
	Latest   LatestVersions   `json:"latest"`
	Versions []VersionSummary `json:"versions"`
}

type LatestVersions struct {
	Release  string `json:"release"`
	Snapshot string `json:"snapshot"`
}

type VersionSummary struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	URL  string `json:"url"`
	SHA1 string `json:"sha1"`
}

func (m *VersionManifest) Find(id string) (*VersionSummary, error) {
	for i := range m.Versions {
		if m.Versions[i].ID == id {
			return &m.Versions[i], nil
		}
	}
	return nil, fmt.Errorf("version %q is not in the manifest", id)
}

type Version struct {
	ID          string              `json:"id"`
	Type        string              `json:"type"`
	MainClass   string              `json:"mainClass"`
	Assets      string              `json:"assets"`
	AssetIndex  AssetIndex          `json:"assetIndex"`
	JavaVersion JavaVersion         `json:"javaVersion"`
	Downloads   map[string]Download `json:"downloads"`
	Libraries   []Library           `json:"libraries"`
	Arguments   Arguments           `json:"arguments"`
}

type AssetIndex struct {
	ID        string `json:"id"`
	SHA1      string `json:"sha1"`
	Size      int64  `json:"size"`
	TotalSize int64  `json:"totalSize"`
	URL       string `json:"url"`
}

type JavaVersion struct {
	Component    string `json:"component"`
	MajorVersion int    `json:"majorVersion"`
}

type Download struct {
	Path string `json:"path,omitempty"`
	SHA1 string `json:"sha1"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

type Library struct {
	Name      string `json:"name"`
	Downloads struct {
		Artifact *Download `json:"artifact,omitempty"`
	} `json:"downloads"`
	Rules []Rule `json:"rules,omitempty"`
}

type Arguments struct {
	Game []Argument `json:"game"`
	JVM  []Argument `json:"jvm"`
}

// Argument is either a bare string or an object carrying rules and one or more
// values. Both shapes appear in the same array.
type Argument struct {
	Rules  []Rule
	Values []string
}

func (a *Argument) UnmarshalJSON(data []byte) error {
	var literal string
	if err := json.Unmarshal(data, &literal); err == nil {
		a.Values = []string{literal}
		return nil
	}

	var conditional struct {
		Rules []Rule          `json:"rules"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(data, &conditional); err != nil {
		return fmt.Errorf("argument is neither a string nor an object: %w", err)
	}

	a.Rules = conditional.Rules

	var many []string
	if err := json.Unmarshal(conditional.Value, &many); err == nil {
		a.Values = many
		return nil
	}
	var one string
	if err := json.Unmarshal(conditional.Value, &one); err != nil {
		return fmt.Errorf("argument value is neither a string nor an array: %w", err)
	}
	a.Values = []string{one}
	return nil
}
