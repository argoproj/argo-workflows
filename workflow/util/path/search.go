package path

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Search(name string) (string, error) {
	if filepath.IsAbs(name) {
		return name, nil
	}
	envPath := os.Getenv("PATH")
	for _, dir := range strings.Split(envPath, ":") {
		absName := filepath.Join(dir, name)
		if _, err := os.Stat(absName); err == nil {
			return absName, nil
		}
	}
	return "", fmt.Errorf("failed to find %s in %s", name, envPath)
}
