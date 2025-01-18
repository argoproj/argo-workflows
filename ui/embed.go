package ui

import "embed"

const EMBED_PATH = "dist/app"

// Embedded contains embedded UI resources
//
//go:embed dist/app
var Embedded embed.FS
