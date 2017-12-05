package argo

import "fmt"

// Version information set by link flags during build
var (
	Version        = "unknown"
	Revision       = "unknown"
	Branch         = "unknown"
	Tag            = ""
	BuildDate      = "unknown"
	FullVersion    = "unknown"
	ImageNamespace = ""
	ImageTag       = Version
)

func init() {
	if ImageNamespace == "" {
		ImageNamespace = "argoproj"
	}
	if Tag != "" {
		// if a git tag was set, use that as our version
		FullVersion = Tag
		ImageTag = Tag
	} else {
		FullVersion = fmt.Sprintf("v%s-%s", Version, Revision[0:7])
	}
}
