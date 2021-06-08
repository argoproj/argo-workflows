package fixtures

import (
	"os"
)

type Need func(s *E2ESuite) (met bool, message string)

var (
	CI Need = func(s *E2ESuite) (bool, string) {
		return os.Getenv("CI") != "", "CI"
	}
	BaseLayerArtifacts Need = func(s *E2ESuite) (bool, string) {
		met, _ := None(K8SAPI, Kubelet)(s)
		return met, "base layer artifact support"
	}
	Docker   = Executor("docker")
	Emissary = Executor("emissary")
	K8SAPI   = Executor("k8sapi")
	Kubelet  = Executor("kubelet")
)

func Executor(e string) Need {
	return func(s *E2ESuite) (bool, string) {
		return s.Config.ContainerRuntimeExecutor == e, e
	}
}

func None(needs ...Need) Need {
	return func(s *E2ESuite) (bool, string) {
		for _, n := range needs {
			met, message := n(s)
			if met {
				return false, "not " + message
			}
		}
		return true, ""
	}
}

func Any(needs ...Need) Need {
	return func(s *E2ESuite) (bool, string) {
		for _, n := range needs {
			met, _ := n(s)
			if met {
				return true, ""
			}
		}
		return false, ""
	}
}

func All(needs ...Need) Need {
	return func(s *E2ESuite) (bool, string) {
		for _, n := range needs {
			met, message := n(s)
			if !met {
				return false, message
			}
		}
		return true, ""
	}
}
