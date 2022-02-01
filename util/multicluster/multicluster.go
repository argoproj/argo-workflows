package multicluster

import (
	"os"
)

// IsEnabled is a flag for whether the new multi-cluster feature is
// enabled
func IsEnabled() bool {
	if os.Getenv("ENABLE_MULTICLUSTER") == "true" {
		return true
	}

	return false
}
