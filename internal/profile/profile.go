// Package profile manages keyword profiles for domain scoring.
package profile

import (
	"fmt"
	"sort"

	"github.com/ul0gic/ctsnare/internal/domain"
)

// Manager loads and manages keyword profiles, satisfying domain.ProfileLoader.
type Manager struct {
	profiles map[string]domain.Profile
}

// NewManager creates a Manager pre-loaded with built-in profiles and any
// custom profiles from config. Custom profiles with an Extends field that
// matches a built-in profile name will inherit the built-in's keywords and
// TLDs, with custom entries appended.
func NewManager(customProfiles map[string]domain.Profile) *Manager {
	m := &Manager{
		profiles: make(map[string]domain.Profile),
	}

	// Register built-in profiles.
	m.profiles["crypto"] = CryptoProfile
	m.profiles["phishing"] = PhishingProfile
	m.profiles["all"] = AllProfile

	// Merge custom profiles. A custom profile can extend a built-in by name.
	for name, custom := range customProfiles {
		resolved := resolveProfile(name, custom, m.profiles)
		m.profiles[name] = resolved
	}

	return m
}

// LoadProfile returns the named profile or an error if it does not exist.
func (m *Manager) LoadProfile(name string) (*domain.Profile, error) {
	p, ok := m.profiles[name]
	if !ok {
		return nil, fmt.Errorf("unknown profile %q; available: %v", name, m.ListProfiles())
	}
	return &p, nil
}

// ListProfiles returns all available profile names in sorted order.
func (m *Manager) ListProfiles() []string {
	names := make([]string, 0, len(m.profiles))
	for name := range m.profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// resolveProfile applies extension logic: if the custom profile's Name field
// contains an "extends:<base>" directive, it inherits keywords, TLDs, and
// skip suffixes from the base profile and appends its own. Otherwise it
// starts fresh.
//
// The extension convention: set the profile Description to "extends:<base>"
// to inherit from a built-in profile. This avoids adding an Extends field
// to the domain.Profile struct (which is frozen).
func resolveProfile(name string, custom domain.Profile, builtins map[string]domain.Profile) domain.Profile {
	// Check if the profile extends a built-in via a convention in Description.
	// Format: "extends:crypto" or "extends:phishing"
	const prefix = "extends:"
	if len(custom.Description) > len(prefix) && custom.Description[:len(prefix)] == prefix {
		baseName := custom.Description[len(prefix):]
		if base, ok := builtins[baseName]; ok {
			return domain.Profile{
				Name:           name,
				Keywords:       mergeUnique(base.Keywords, custom.Keywords),
				SuspiciousTLDs: mergeUnique(base.SuspiciousTLDs, custom.SuspiciousTLDs),
				SkipSuffixes:   mergeUnique(base.SkipSuffixes, custom.SkipSuffixes),
				Description:    fmt.Sprintf("Custom profile extending %s", baseName),
			}
		}
	}

	// No extension -- use as-is, filling in name if empty.
	result := custom
	if result.Name == "" {
		result.Name = name
	}
	return result
}
