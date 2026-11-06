package executor

import "github.com/argoproj/argo-workflows/v4/workflow/executor"

// PodPaths returns the paths of the task a classic pod runs, with its
// shared directory at varRunArgo: /var/run/argo in a pod, elsewhere in
// tests. Pod commands build their paths here so that directory has one
// source.
func PodPaths(varRunArgo string) executor.TaskPaths {
	paths := executor.PodPaths()
	paths.VarRunArgo = varRunArgo
	return paths
}
