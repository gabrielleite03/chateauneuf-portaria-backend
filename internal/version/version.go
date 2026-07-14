package version

var (
	Version = "dev"
	Commit  = "unknown"
)

type Info struct {
	Service string `json:"service"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

func Current() Info {
	return Info{
		Service: "backend",
		Version: Version,
		Commit:  Commit,
	}
}
