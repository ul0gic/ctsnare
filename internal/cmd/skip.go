package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ul0gic/ctsnare/internal/config"
	"github.com/ul0gic/ctsnare/internal/profile"
)

var skipCmd = &cobra.Command{
	Use:   "skip",
	Short: "Manage domain skip suffix whitelist",
	Long: `Manage the list of domain suffixes excluded from scoring.

The skip list prevents infrastructure platforms (CDNs, cloud hosts, big tech)
from flooding results with noise. It is built in three layers:

  1. Global (hardcoded) -- cloud providers, CDNs, PaaS, big tech infra
  2. User additions     -- extra domains you want to skip
  3. User removals      -- globals you want to un-skip (to monitor them)

Effective skip list = globals + additions - removals

Examples:
  ctsnare skip list
  ctsnare skip add sailpoint.com jpmchase.net
  ctsnare skip remove google.com
  ctsnare skip reset --confirm`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

var skipListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show effective skip list",
	Long: `Display the full effective skip list grouped by category:
globals (hardcoded), user additions, and user removals.

Examples:
  ctsnare skip list
  ctsnare skip list --config ~/.config/ctsnare/config.toml`,
	RunE: runSkipList,
}

var skipAddCmd = &cobra.Command{
	Use:   "add <domain> [domain...]",
	Short: "Add domains to the skip list",
	Long: `Add one or more domain suffixes to the user skip list.

Added domains are persisted to the config file and will be excluded
from scoring on the next run of 'ctsnare watch'.

If a domain is already in the global (hardcoded) skip list, a warning
is printed. Domains are normalized to lowercase.

Examples:
  ctsnare skip add example.com
  ctsnare skip add internal-corp.net staging.example.org`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSkipAdd,
}

var skipRemoveCmd = &cobra.Command{
	Use:   "remove <domain> [domain...]",
	Short: "Remove domains from the skip list",
	Long: `Remove one or more domain suffixes from the skip list.

If the domain is a user addition, it is removed from the additions list.
If the domain is a global (hardcoded) suffix, it is added to the removals
list, effectively "un-skipping" it so it will be scored again.

Examples:
  ctsnare skip remove example.com
  ctsnare skip remove google.com`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSkipRemove,
}

var skipResetConfirm bool

var skipResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Clear all user overrides",
	Long: `Reset the skip list to globals only by clearing all user additions
and removals from the config file.

Requires --confirm to prevent accidental resets.

Examples:
  ctsnare skip reset --confirm`,
	RunE: runSkipReset,
}

func init() {
	skipResetCmd.Flags().BoolVar(&skipResetConfirm, "confirm", false, "required: confirm reset to prevent accidents")

	skipCmd.AddCommand(skipListCmd)
	skipCmd.AddCommand(skipAddCmd)
	skipCmd.AddCommand(skipRemoveCmd)
	skipCmd.AddCommand(skipResetCmd)
	rootCmd.AddCommand(skipCmd)
}

// skipConfigPath returns the config file path to use for skip commands.
// Prefers --config flag, falls back to DefaultConfigPath.
func skipConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}
	return config.DefaultConfigPath()
}

// runSkipList displays the effective skip list grouped by category.
func runSkipList(_ *cobra.Command, _ []string) error {
	path := skipConfigPath()
	overrides, err := config.LoadSkipOverrides(path)
	if err != nil {
		return fmt.Errorf("loading skip overrides: %w", err)
	}

	globals := profile.GlobalSkipSuffixes
	effective := config.MergeSkipSuffixes(globals, overrides)

	// Build lookup sets for display categorization.
	removalsSet := toSet(overrides.Removals)
	globalsSet := toSet(globals)

	fmt.Printf("Skip Suffix List (effective: %d domains)\n\n", len(effective))

	fmt.Printf("Global (hardcoded): %d\n", countActive(globals, removalsSet))
	for _, s := range globals {
		if _, removed := removalsSet[s]; removed {
			continue
		}
		fmt.Printf("  %s\n", s)
	}
	fmt.Println()

	if len(overrides.Additions) > 0 {
		fmt.Printf("User additions: %d\n", len(overrides.Additions))
		for _, s := range overrides.Additions {
			if _, isGlobal := globalsSet[s]; isGlobal {
				fmt.Printf("  %s (already global)\n", s)
			} else {
				fmt.Printf("  %s\n", s)
			}
		}
		fmt.Println()
	}

	if len(overrides.Removals) > 0 {
		fmt.Printf("User removals (overriding globals): %d\n", len(overrides.Removals))
		for _, s := range overrides.Removals {
			fmt.Printf("  %s\n", s)
		}
		fmt.Println()
	}

	fmt.Printf("Config: %s\n", path)
	return nil
}

// runSkipAdd adds domains to the user additions list.
func runSkipAdd(_ *cobra.Command, args []string) error {
	path := skipConfigPath()
	overrides, err := config.LoadSkipOverrides(path)
	if err != nil {
		return fmt.Errorf("loading skip overrides: %w", err)
	}

	globalsSet := toSet(profile.GlobalSkipSuffixes)
	additionsSet := toSet(overrides.Additions)

	var added []string
	for _, raw := range args {
		domain, validateErr := validateDomain(raw)
		if validateErr != nil {
			fmt.Fprintf(os.Stderr, "Skipping %q: %s\n", raw, validateErr)
			continue
		}

		// Warn if already a global.
		if _, isGlobal := globalsSet[domain]; isGlobal {
			fmt.Fprintf(os.Stderr, "Warning: %q is already in the global skip list.\n", domain)
			continue
		}

		// Skip if already in additions.
		if _, exists := additionsSet[domain]; exists {
			fmt.Fprintf(os.Stderr, "Already in additions: %q\n", domain)
			continue
		}

		// If the domain was previously removed (un-skipped), undo that removal.
		overrides.Removals = removeFromSlice(overrides.Removals, domain)

		overrides.Additions = append(overrides.Additions, domain)
		additionsSet[domain] = struct{}{}
		added = append(added, domain)
	}

	if len(added) == 0 {
		return nil
	}

	if err := config.SaveSkipOverrides(path, overrides); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Added to skip list: %s\n", strings.Join(added, ", "))
	return nil
}

// runSkipRemove removes domains from the skip list.
func runSkipRemove(_ *cobra.Command, args []string) error {
	path := skipConfigPath()
	overrides, err := config.LoadSkipOverrides(path)
	if err != nil {
		return fmt.Errorf("loading skip overrides: %w", err)
	}

	globalsSet := toSet(profile.GlobalSkipSuffixes)
	additionsSet := toSet(overrides.Additions)

	var removed []string
	for _, raw := range args {
		domain, validateErr := validateDomain(raw)
		if validateErr != nil {
			fmt.Fprintf(os.Stderr, "Skipping %q: %s\n", raw, validateErr)
			continue
		}

		// Case 1: domain is a user addition -- remove from additions.
		if _, inAdditions := additionsSet[domain]; inAdditions {
			overrides.Additions = removeFromSlice(overrides.Additions, domain)
			delete(additionsSet, domain)
			removed = append(removed, domain)
			fmt.Printf("Removed from user additions: %s\n", domain)
			continue
		}

		// Case 2: domain is a global -- add to removals (un-skip it).
		if _, isGlobal := globalsSet[domain]; isGlobal {
			// Check if already in removals.
			removalsSet := toSet(overrides.Removals)
			if _, already := removalsSet[domain]; already {
				fmt.Fprintf(os.Stderr, "Already un-skipped: %q\n", domain)
				continue
			}
			overrides.Removals = append(overrides.Removals, domain)
			removed = append(removed, domain)
			fmt.Printf("Un-skipped global: %s (will now be scored)\n", domain)
			continue
		}

		fmt.Fprintf(os.Stderr, "Warning: %q is not in any skip list.\n", domain)
	}

	if len(removed) == 0 {
		return nil
	}

	if err := config.SaveSkipOverrides(path, overrides); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	return nil
}

// runSkipReset clears all user overrides.
func runSkipReset(_ *cobra.Command, _ []string) error {
	if !skipResetConfirm {
		path := skipConfigPath()
		overrides, err := config.LoadSkipOverrides(path)
		if err != nil {
			return fmt.Errorf("loading skip overrides: %w", err)
		}

		total := len(overrides.Additions) + len(overrides.Removals)
		fmt.Fprintf(os.Stderr, "You have %d user overrides (%d additions, %d removals).\n",
			total, len(overrides.Additions), len(overrides.Removals))
		fmt.Fprintln(os.Stderr, "Run with --confirm to reset all overrides to defaults.")
		return nil
	}

	path := skipConfigPath()
	empty := config.SkipOverrides{
		Additions: []string{},
		Removals:  []string{},
	}

	if err := config.SaveSkipOverrides(path, empty); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println("All user skip overrides cleared. Skip list reset to globals only.")
	return nil
}

// validateDomain checks that a string looks like a valid domain suffix.
// Returns the normalized (lowercase, trimmed) domain or an error.
func validateDomain(raw string) (string, error) {
	d := strings.TrimSpace(strings.ToLower(raw))

	// Remove trailing dot if present.
	d = strings.TrimSuffix(d, ".")

	if d == "" {
		return "", fmt.Errorf("empty domain")
	}
	if strings.Contains(d, "://") {
		return "", fmt.Errorf("must not contain protocol prefix (remove http:// or https://)")
	}
	if !strings.Contains(d, ".") {
		return "", fmt.Errorf("must contain at least one dot")
	}
	if strings.ContainsAny(d, " \t\n") {
		return "", fmt.Errorf("must not contain whitespace")
	}

	return d, nil
}

// toSet converts a string slice to a set (map[string]struct{}).
func toSet(items []string) map[string]struct{} {
	s := make(map[string]struct{}, len(items))
	for _, item := range items {
		s[item] = struct{}{}
	}
	return s
}

// removeFromSlice returns a new slice with all occurrences of target removed.
func removeFromSlice(s []string, target string) []string {
	result := make([]string, 0, len(s))
	for _, item := range s {
		if item != target {
			result = append(result, item)
		}
	}
	return result
}

// countActive counts items in a slice that are not in the excluded set.
func countActive(items []string, excluded map[string]struct{}) int {
	count := 0
	for _, item := range items {
		if _, ex := excluded[item]; !ex {
			count++
		}
	}
	return count
}
