package manifest

import "time"

const SchemaVersion = 1

type Source string

const (
	SourceModrinth   Source = "modrinth"
	SourceCurseForge Source = "curseforge"
	SourceDirect     Source = "direct"
)

type AuthMode string

const (
	AuthMicrosoft AuthMode = "microsoft"
	AuthOffline   AuthMode = "offline"
)

type LoaderKind string

const (
	LoaderVanilla  LoaderKind = "vanilla"
	LoaderFabric   LoaderKind = "fabric"
	LoaderQuilt    LoaderKind = "quilt"
	LoaderNeoForge LoaderKind = "neoforge"
	LoaderForge    LoaderKind = "forge"
)

type GPUVendor string

const (
	GPUAuto   GPUVendor = "auto"
	GPUAMD    GPUVendor = "amd"
	GPUIntel  GPUVendor = "intel"
	GPUNvidia GPUVendor = "nvidia"
)

// Manifest is the complete description of one bootable configuration. It is the
// only artifact PhantomMC servers store and serve.
type Manifest struct {
	SchemaVersion int       `json:"schemaVersion"`
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"createdAt"`

	Minecraft Minecraft `json:"minecraft"`
	Loader    Loader    `json:"loader"`
	Java      Java      `json:"java"`
	Auth      Auth      `json:"auth"`
	Graphics  Graphics  `json:"graphics"`
	JVM       JVM       `json:"jvm"`
	Mods      []Mod     `json:"mods,omitempty"`
	Servers   []Server  `json:"servers,omitempty"`
}

type Minecraft struct {
	Version string `json:"version"`
}

type Loader struct {
	Kind    LoaderKind `json:"kind"`
	Version string     `json:"version,omitempty"`
}

type Java struct {
	Major        int    `json:"major"`
	Distribution string `json:"distribution"`
}

type Auth struct {
	Mode            AuthMode     `json:"mode"`
	OfflineUsername string       `json:"offlineUsername,omitempty"`
	Entitlement     *Entitlement `json:"entitlement,omitempty"`
}

// Entitlement is issued by the web builder after a successful ownership check.
// It carries no token and no account identifier, only the fact that ownership
// was confirmed and when that expires. See ADR 0006.
type Entitlement struct {
	IssuedAt  time.Time `json:"issuedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	Signature string    `json:"signature"`
}

type Graphics struct {
	Vendor       GPUVendor `json:"vendor"`
	UpscaleTo    string    `json:"upscaleTo,omitempty"`
	RenderWidth  int       `json:"renderWidth,omitempty"`
	RenderHeight int       `json:"renderHeight,omitempty"`
}

type JVM struct {
	HeapMB int      `json:"heapMB"`
	Args   []string `json:"args,omitempty"`
}

type Mod struct {
	Source    Source   `json:"source"`
	ProjectID string   `json:"projectId"`
	VersionID string   `json:"versionId"`
	Filename  string   `json:"filename"`
	Artifact  Artifact `json:"artifact"`

	// Manual is set when the upstream forbids third party download. The runtime
	// prompts for the file instead of fetching it. See ADR 0004.
	Manual bool `json:"manual,omitempty"`
}

type Artifact struct {
	URL    string `json:"url,omitempty"`
	Size   int64  `json:"size"`
	SHA512 string `json:"sha512"`
}

type Server struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (m *Manifest) RequiresNetwork() bool {
	if len(m.Mods) > 0 {
		return true
	}
	return m.Auth.Mode == AuthMicrosoft
}

func (e *Entitlement) Valid(now time.Time) bool {
	if e == nil || e.Signature == "" {
		return false
	}
	return now.Before(e.ExpiresAt)
}
