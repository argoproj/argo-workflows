package multicluster

import (
	"os"
	"strconv"
)

// IsEnabled is a flag for whether the new multi-cluster feature is
// enabled
func IsEnabled() bool {
	isEnabled, err := strconv.ParseBool(os.Getenv("ENABLE_MULTICLUSTER"))
	if err != nil {
		return false
	}

	return isEnabled
}
