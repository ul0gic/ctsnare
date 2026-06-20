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

// NewManager pre-loads the built-in profiles plus any custom profiles, which may
// extend a built-in by name (see resolveProfile).
func NewManager(customProfiles map[string]domain.Profile) *Manager {
	m := &Manager{
		profiles: make(map[string]domain.Profile),
	}

	m.profiles["crypto"] = CryptoProfile
	m.profiles["phishing"] = PhishingProfile
	m.profiles["ai"] = AIProfile
	m.profiles["all"] = AllProfile

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

// resolveProfile inherits from a built-in when custom.Description is "extends:<base>";
// the convention lives in Description to avoid adding a field to the frozen domain.Profile.
func resolveProfile(name string, custom domain.Profile, builtins map[string]domain.Profile) domain.Profile {
	const prefix = "extends:"
	if len(custom.Description) > len(prefix) && custom.Description[:len(prefix)] == prefix {
		baseName := custom.Description[len(prefix):]
		if base, ok := builtins[baseName]; ok {
			return domain.Profile{
				Name:           name,
				BrandKeywords:  mergeUnique(base.BrandKeywords, custom.BrandKeywords),
				Keywords:       mergeUnique(base.Keywords, custom.Keywords),
				SuspiciousTLDs: mergeUnique(base.SuspiciousTLDs, custom.SuspiciousTLDs),
				SkipSuffixes:   mergeUnique(base.SkipSuffixes, custom.SkipSuffixes),
				Description:    "Custom profile extending " + baseName,
			}
		}
	}

	result := custom
	if result.Name == "" {
		result.Name = name
	}
	return result
}
