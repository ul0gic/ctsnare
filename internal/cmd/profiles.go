package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ul0gic/ctsnare/internal/domain"
)

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "List and inspect keyword profiles",
	Long: `List all available keyword profiles or inspect a specific profile's
keywords, suspicious TLDs, and skip suffixes.`,
	RunE: runProfilesList,
}

var profilesShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show details of a specific profile",
	Long:  `Display the full details of a keyword profile including keywords, suspicious TLDs, and skip suffixes.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProfilesShow,
}

func init() {
	profilesCmd.AddCommand(profilesShowCmd)
	rootCmd.AddCommand(profilesCmd)
}

// runProfilesList is the placeholder RunE for listing profiles.
// Real profile manager wiring happens in Phase 3.
func runProfilesList(_ *cobra.Command, _ []string) error {
	// Phase 3 will wire: config loading, profile manager creation, list profiles.
	// For now, show the known built-in profile names.
	builtins := []struct {
		name string
		desc string
	}{
		{"all", "Combined crypto + phishing keywords and TLDs"},
		{"crypto", "Cryptocurrency, DeFi, and exchange-related keywords"},
		{"phishing", "Login, credential theft, and brand impersonation keywords"},
	}

	fmt.Println("Available Profiles:")
	fmt.Println()
	for _, p := range builtins {
		fmt.Printf("  %-12s %s\n", p.name, p.desc)
	}
	return nil
}

// runProfilesShow is the placeholder RunE for showing profile details.
// Real profile manager wiring happens in Phase 3.
func runProfilesShow(_ *cobra.Command, args []string) error {
	name := args[0]
	// Phase 3 will wire: config loading, profile manager, LoadProfile call.
	// For now, return an error indicating Phase 3 wiring is needed.
	return fmt.Errorf("profile %q details not yet available -- integration happens in Phase 3", name)
}

// PrintProfileDetail prints the full details of a profile to stdout.
// Used by the profiles show command after Phase 3 wiring.
func PrintProfileDetail(p *domain.Profile) {
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

	fmt.Printf("Skip Suffixes (%d):\n", len(p.SkipSuffixes))
	for _, s := range p.SkipSuffixes {
		fmt.Printf("  - %s\n", s)
	}
}
