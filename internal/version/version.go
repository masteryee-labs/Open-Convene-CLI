// Package version holds the build-time version string for openconvene.
//
// The version is injected via ldflags:
//
//	go build -ldflags "-X github.com/masteryee-labs/open-convene-cli/internal/version.Version=v1.0.0" \
//	          -o openconvene ./cmd/openconvene
//
// If not set at build time, Version defaults to "dev".
package version

// Version is the semantic version of the openconvene binary.
// It is set at build time via -ldflags "-X ...version.Version=vX.Y.Z".
var Version = "dev"
