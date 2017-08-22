package cluster

import "applatix.io/axerror"

// Add lock later
var ClusterSettings map[string]string = map[string]string{}

const (
	PublicReadEnabledKey = "public_read_enabled"
)

func InitClusterSettings() *axerror.AXError {
	settings, axErr := GetClusterSettings(nil)
	if axErr != nil {
		return axErr
	}

	for _, setting := range settings {
		ClusterSettings[setting.Key] = setting.Value
	}

	return nil
}
