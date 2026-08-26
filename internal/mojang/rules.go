package mojang

import "regexp"

const (
	ActionAllow    = "allow"
	ActionDisallow = "disallow"
)

type Rule struct {
	Action   string          `json:"action"`
	OS       *OSCondition    `json:"os,omitempty"`
	Features map[string]bool `json:"features,omitempty"`
}

type OSCondition struct {
	Name    string `json:"name,omitempty"`
	Arch    string `json:"arch,omitempty"`
	Version string `json:"version,omitempty"`
}

// Platform describes the machine a launch is being prepared for, using Mojang's
// vocabulary rather than Go's.
type Platform struct {
	OS       string
	Arch     string
	Version  string
	Features map[string]bool
}

func LinuxAMD64() Platform {
	return Platform{OS: "linux", Arch: "x86_64"}
}

// Allows reports whether a rule set permits something on this platform. An
// empty rule set allows. Otherwise the default is deny and the last matching
// rule wins, which is the behaviour the official launcher implements.
func (p Platform) Allows(rules []Rule) bool {
	if len(rules) == 0 {
		return true
	}

	allowed := false
	for _, rule := range rules {
		if !p.matches(rule) {
			continue
		}
		allowed = rule.Action == ActionAllow
	}
	return allowed
}

func (p Platform) matches(rule Rule) bool {
	if rule.OS != nil {
		if rule.OS.Name != "" && rule.OS.Name != p.OS {
			return false
		}
		if rule.OS.Arch != "" && rule.OS.Arch != p.Arch {
			return false
		}
		if rule.OS.Version != "" {
			matched, err := regexp.MatchString(rule.OS.Version, p.Version)
			if err != nil || !matched {
				return false
			}
		}
	}

	for feature, want := range rule.Features {
		if p.Features[feature] != want {
			return false
		}
	}
	return true
}
