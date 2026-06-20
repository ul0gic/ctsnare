package cmd

import "runtime/debug"

// version is injected by GoReleaser via ldflags; it stays "dev" for plain
// builds, where resolveVersion falls back to module build info.
var version = "dev"

// resolveVersion returns the effective version string for --version output.
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}
