package packr

import (
	"go/build"
	"os"
	"strings"
)

// GoPath returns the current GOPATH env var
// or if it's missing, the default.
func GoPath() string {
	go_path := strings.Split(os.Getenv("GOPATH"), string(os.PathListSeparator))
	if len(go_path) == 0 || go_path[0] == "" {
		return build.Default.GOPATH
	}
	return go_path[0]
}

// GoBin returns the current GO_BIN env var
// or if it's missing, a default of "go"
func GoBin() string {
	go_bin := os.Getenv("GO_BIN")
	if go_bin == "" {
		return "go"
	}
	return go_bin
}
