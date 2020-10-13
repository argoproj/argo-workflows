package fixtures

import (
	"os"

	"github.com/argoproj/argo/workflow/common"
)

type Cap string

const (
	RunAsNonRoot    Cap = "RunAsNonRoot"
	BaseLayerOutput Cap = "BaseLayerOutput"
)

var supportedCaps = map[string]map[Cap]bool{
	common.ContainerRuntimeExecutorDocker:  {BaseLayerOutput: true},
	common.ContainerRuntimeExecutorK8sAPI:  {RunAsNonRoot: true},
	common.ContainerRuntimeExecutorKubelet: {RunAsNonRoot: true},
	// base layer output does not work on CI
	common.ContainerRuntimeExecutorPNS: {RunAsNonRoot: true, BaseLayerOutput: os.Getenv("CI") != "true"},
}
