package version

var (
	// Version is the main version number that is being run at the moment.
	Version = "0.1.0"

	// Commit is the git commit that was compiled. This will be filled by the compiler.
	Commit = "none"

	// BuildTime is the human-readable time when the binary was built. This will be filled by the compiler.
	BuildTime = "unknown"
)
