package argo

import "fmt"

// Version information set by link flags during build
var (
	Version        = "unknown"
	Revision       = "unknown"
	Branch         = "unknown"
	Tag            = ""
	BuildDate      = "unknown"
	FullVersion    = fmt.Sprintf("%s-%s", Version, Revision)
	DisplayVersion = fmt.Sprintf("%s (Build Date: %s)", FullVersion, BuildDate)
)
