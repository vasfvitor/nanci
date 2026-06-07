package buildinfo

var (
	// Version is injected at build time (e.g. via -ldflags "-X '...Version=...'")
	Version = "0.0.0-dev"

	// Commit is the Git commit hash injected at build time
	Commit = "unknown"

	// Date is the build date injected at build time
	Date = "unknown"
)
