package manifest

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	ErrSchemaVersion = errors.New("unsupported schema version")
	ErrNoEntitlement = errors.New("offline mode requires a valid entitlement")
)

var (
	idPattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,63}$`)
	usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,16}$`)
	sha512Pattern   = regexp.MustCompile(`^[a-f0-9]{128}$`)
)

type ValidationError struct {
	Field  string
	Reason string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	parts := make([]string, len(e))
	for i, v := range e {
		parts[i] = v.Error()
	}
	return strings.Join(parts, "; ")
}

func (m *Manifest) Validate(now time.Time) error {
	var errs ValidationErrors
	add := func(field, reason string) {
		errs = append(errs, ValidationError{Field: field, Reason: reason})
	}

	if m.SchemaVersion != SchemaVersion {
		add("schemaVersion", fmt.Sprintf("expected %d, got %d", SchemaVersion, m.SchemaVersion))
	}
	if !idPattern.MatchString(m.ID) {
		add("id", "must be 3 to 64 lowercase alphanumeric or hyphen characters")
	}
	if strings.TrimSpace(m.Name) == "" {
		add("name", "required")
	}
	if strings.TrimSpace(m.Minecraft.Version) == "" {
		add("minecraft.version", "required")
	}

	switch m.Loader.Kind {
	case LoaderVanilla:
		if m.Loader.Version != "" {
			add("loader.version", "must be empty for vanilla")
		}
	case LoaderFabric, LoaderQuilt, LoaderForge, LoaderNeoForge:
		if m.Loader.Version == "" {
			add("loader.version", "required for "+string(m.Loader.Kind))
		}
	default:
		add("loader.kind", "unknown loader")
	}

	if m.Loader.Kind == LoaderVanilla && len(m.Mods) > 0 {
		add("mods", "vanilla loader cannot load mods")
	}

	if m.Java.Major < 8 {
		add("java.major", "must be 8 or greater")
	}
	if m.JVM.HeapMB < 512 {
		add("jvm.heapMB", "must be at least 512")
	}

	switch m.Graphics.Vendor {
	case GPUAuto, GPUAMD, GPUIntel, GPUNvidia:
	default:
		add("graphics.vendor", "unknown vendor")
	}

	errs = append(errs, validateAuth(&m.Auth, now)...)
	errs = append(errs, validateMods(m.Mods)...)
	errs = append(errs, validateServers(m.Servers)...)

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func validateAuth(a *Auth, now time.Time) ValidationErrors {
	var errs ValidationErrors
	add := func(field, reason string) {
		errs = append(errs, ValidationError{Field: field, Reason: reason})
	}

	switch a.Mode {
	case AuthMicrosoft:
		if a.OfflineUsername != "" {
			add("auth.offlineUsername", "must be empty in microsoft mode")
		}
		if strings.TrimSpace(a.ClientID) == "" {
			add("auth.clientId", "required in microsoft mode")
		}
	case AuthOffline:
		if a.ClientID != "" {
			add("auth.clientId", "must be empty in offline mode")
		}
		if !usernamePattern.MatchString(a.OfflineUsername) {
			add("auth.offlineUsername", "must be 3 to 16 characters of A-Z, a-z, 0-9 or underscore")
		}
		if !a.Entitlement.Valid(now) {
			add("auth.entitlement", ErrNoEntitlement.Error())
		}
	default:
		add("auth.mode", "must be microsoft or offline")
	}
	return errs
}

func validateMods(mods []Mod) ValidationErrors {
	var errs ValidationErrors
	seen := make(map[string]int, len(mods))

	for i, mod := range mods {
		field := fmt.Sprintf("mods[%d]", i)

		switch mod.Source {
		case SourceModrinth, SourceCurseForge, SourceDirect:
		default:
			errs = append(errs, ValidationError{field + ".source", "unknown source"})
		}

		if mod.Filename == "" || strings.ContainsAny(mod.Filename, `/\`) {
			errs = append(errs, ValidationError{field + ".filename", "must be a bare filename"})
		}
		if !strings.HasSuffix(mod.Filename, ".jar") {
			errs = append(errs, ValidationError{field + ".filename", "must end in .jar"})
		}

		if prev, dup := seen[mod.Filename]; dup {
			errs = append(errs, ValidationError{
				field + ".filename",
				fmt.Sprintf("duplicate of mods[%d]", prev),
			})
		}
		seen[mod.Filename] = i

		errs = append(errs, validateArtifact(field+".artifact", mod.Artifact, mod.Manual)...)
	}
	return errs
}

func validateArtifact(field string, a Artifact, manual bool) ValidationErrors {
	var errs ValidationErrors

	if !sha512Pattern.MatchString(a.SHA512) {
		errs = append(errs, ValidationError{field + ".sha512", "must be a lowercase hex sha512 digest"})
	}
	if a.Size <= 0 {
		errs = append(errs, ValidationError{field + ".size", "must be positive"})
	}

	if manual {
		if a.URL != "" {
			errs = append(errs, ValidationError{field + ".url", "must be empty when manual"})
		}
		return errs
	}

	parsed, err := url.Parse(a.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		errs = append(errs, ValidationError{field + ".url", "must be an absolute https url"})
	}
	return errs
}

func validateServers(servers []Server) ValidationErrors {
	var errs ValidationErrors
	for i, s := range servers {
		field := fmt.Sprintf("servers[%d]", i)
		if strings.TrimSpace(s.Name) == "" {
			errs = append(errs, ValidationError{field + ".name", "required"})
		}
		if strings.TrimSpace(s.Address) == "" || strings.ContainsAny(s.Address, " \t") {
			errs = append(errs, ValidationError{field + ".address", "must be a host or host:port"})
		}
	}
	return errs
}
