package fixtures

import (
	"os"
)

type Need func(s *E2ESuite) (met bool, message string)

var (
	TODO Need = func(s *E2ESuite) (bool, string) {
		return false, "something needs to be done"
	}
	RBAC Need = func(s *E2ESuite) (bool, string) {
		return os.Getenv("CI") != "", "RBAC (and therefore CI)"
	}
	Offloading Need = func(s *E2ESuite) (bool, string) {
		return s.Persistence.IsEnabled(), "offloading"
	}
	WorkflowArchive = Offloading
	Docker          = Executor("docker")
	K8SAPI          = Executor("k8sapi")
	Kubelet         = Executor("kubelet")
	PNS             = Executor("pns")
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
