package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ul0gic/ctsnare/internal/config"
	"github.com/ul0gic/ctsnare/internal/domain"
	"github.com/ul0gic/ctsnare/internal/profile"
)

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "List and inspect keyword profiles",
	Long: `List all available keyword profiles or inspect a specific profile's
keywords, suspicious TLDs, and skip suffixes.

Built-in profiles: crypto, phishing, all.
Custom profiles are loaded from the config file (--config).

Examples:
  ctsnare profiles
  ctsnare profiles show crypto
  ctsnare profiles show all`,
	RunE: runProfilesList,
}

var profilesShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show full details of a keyword profile",
	Long: `Display the full details of a keyword profile: keywords, suspicious TLDs,
and the effective skip suffix list (globals + user additions - user removals).

Examples:
  ctsnare profiles show crypto
  ctsnare profiles show phishing
  ctsnare profiles show all`,
	Args: cobra.ExactArgs(1),
	RunE: runProfilesShow,
}

func init() {
	profilesCmd.AddCommand(profilesShowCmd)
	rootCmd.AddCommand(profilesCmd)
}

// newProfileManager creates a profile.Manager wired with custom profiles from config.
func newProfileManager() (*profile.Manager, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return profile.NewManager(cfg.CustomProfiles), nil
}

// runProfilesList lists all available profiles with their descriptions.
func runProfilesList(_ *cobra.Command, _ []string) error {
	mgr, err := newProfileManager()
	if err != nil {
		return err
	}

	names := mgr.ListProfiles()
	fmt.Println("Available Profiles:")
	fmt.Println()
	for _, name := range names {
		p, loadErr := mgr.LoadProfile(name)
		if loadErr != nil {
			continue
		}
		desc := p.Description
		if desc == "" {
			desc = fmt.Sprintf("%d keywords", len(p.Keywords))
		}
		fmt.Printf("  %-12s %s\n", name, desc)
	}
	return nil
}

// runProfilesShow displays full details for a named profile, including the
// effective skip suffix list with user overrides applied.
func runProfilesShow(_ *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	mgr := profile.NewManager(cfg.CustomProfiles)
	p, err := mgr.LoadProfile(args[0])
	if err != nil {
		return err
	}

	// Compute effective skip list and inject into profile for display.
	p.SkipSuffixes = config.MergeSkipSuffixes(profile.GlobalSkipSuffixes, cfg.SkipOverrides)

	PrintProfileDetail(p, cfg.SkipOverrides)
	return nil
}

// PrintProfileDetail prints the full details of a profile to stdout.
// If overrides has additions or removals, annotates skip suffixes accordingly.
func PrintProfileDetail(p *domain.Profile, overrides config.SkipOverrides) {
	fmt.Printf("Profile: %s\n", p.Name)
	if p.Description != "" {
		fmt.Printf("Description: %s\n", p.Description)
	}
	fmt.Println()

	fmt.Printf("Keywords (%d):\n", len(p.Keywords))
	fmt.Printf("  %s\n", strings.Join(p.Keywords, ", "))
	fmt.Println()

	fmt.Printf("Suspicious TLDs (%d):\n", len(p.SuspiciousTLDs))
	fmt.Printf("  %s\n", strings.Join(p.SuspiciousTLDs, ", "))
	fmt.Println()

	// Build lookup sets for annotating skip suffixes.
	additionsSet := make(map[string]struct{}, len(overrides.Additions))
	for _, a := range overrides.Additions {
		additionsSet[a] = struct{}{}
	}

	fmt.Printf("Skip Suffixes (%d effective):\n", len(p.SkipSuffixes))
	for _, s := range p.SkipSuffixes {
		if _, isUserAdded := additionsSet[s]; isUserAdded {
			fmt.Printf("  - %s [+]\n", s)
		} else {
			fmt.Printf("  - %s\n", s)
		}
	}

	// Show removed globals if any.
	if len(overrides.Removals) > 0 {
		fmt.Printf("\nRemoved from globals (will be scored): %d\n", len(overrides.Removals))
		for _, r := range overrides.Removals {
			fmt.Printf("  - %s [-]\n", r)
		}
	}
}
