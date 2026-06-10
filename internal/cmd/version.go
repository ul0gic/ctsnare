package cmd

import "runtime/debug"

// version is the release version, injected by GoReleaser via ldflags
// (-X .../internal/cmd.version={{.Version}}). It stays "dev" for plain
// `go build`; for `go install module@version` builds resolveVersion falls
// back to the module version recorded in build info.
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
