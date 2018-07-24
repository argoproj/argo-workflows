package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// Clean up an *-packr.go files
func Clean(root string) {
	root, _ = filepath.EvalSymlinks(root)
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		base := filepath.Base(path)
		if base == ".git" || base == "vendor" || base == "node_modules" {
			return filepath.SkipDir
		}
		if info == nil || info.IsDir() {
			return nil
		}
		if strings.Contains(base, "-packr.go") {
			err := os.Remove(path)
			if err != nil {
				fmt.Println(err)
				return errors.WithStack(err)
			}
		}
		return nil
	})
}
