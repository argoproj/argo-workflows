// +build !windows

package parser

import (
	"fmt"
	"os"
	"path"
	"strings"
)

func getPkgPath(fname string, isDir bool) (string, error) {
	if !path.IsAbs(fname) {
		pwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		fname = path.Join(pwd, fname)
	}

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		var err error
		gopath, err = getDefaultGoPath()
		if err != nil {
			return "", fmt.Errorf("cannot determine GOPATH: %s", err)
		}
	}

	for _, p := range strings.Split(os.Getenv("GOPATH"), ":") {
		prefix := path.Join(p, "src") + "/"
		if rel := strings.TrimPrefix(fname, prefix); rel != fname {
			if !isDir {
				return path.Dir(rel), nil
			} else {
				return path.Clean(rel), nil
			}
		}
	}

	return "", fmt.Errorf("file '%v' is not in GOPATH", fname)
}
