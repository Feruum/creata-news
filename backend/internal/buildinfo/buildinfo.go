package buildinfo

// Version is updated by release-please and may be overridden at link time.
var Version = "1.0.0" // x-release-please-version

var (
	Commit  = "unknown"
	BuiltAt = "unknown"
)

type Info struct {
	Service     string `json:"service"`
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	BuiltAt     string `json:"built_at"`
	Environment string `json:"environment"`
}

func Current(service, environment string) Info {
	return Info{
		Service:     service,
		Version:     Version,
		Commit:      Commit,
		BuiltAt:     BuiltAt,
		Environment: environment,
	}
}
