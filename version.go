package argo

import "fmt"

// Version information set by link flags during build
var (
	Version        = "unknown"
	Revision       = "unknown"
	Branch         = "unknown"
	Tag            = ""
	BuildDate      = "unknown"
	ShortRevision  = Revision[0:7]
	FullVersion    = fmt.Sprintf("%s-%s", Version, ShortRevision)
	DisplayVersion = fmt.Sprintf("%s (Build Date: %s)", FullVersion, BuildDate)
)
