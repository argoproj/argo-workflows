package fixtures

import (
	"os"
)

type Need func(s *E2ESuite) bool

var (
	TODO Need = func(s *E2ESuite) bool {
		return false // something needs to be done, so this need is always unmet
	}
	CI Need = func(s *E2ESuite) bool {
		return os.Getenv("CI") != ""
	}
	Offloading Need = func(s *E2ESuite) bool {
		return s.Persistence.IsEnabled()
	}
	Docker = Executor("docker")
	K8SAPI = Executor("k8sapi")
	PNS    = Executor("pns")
)

func Executor(e string) Need {
	return func(s *E2ESuite) bool {
		return s.Config.ContainerRuntimeExecutor == e
	}
}

func Not(n Need) Need {
	return func(s *E2ESuite) bool {
		return !n(s)
	}
}

func Or(needs ...Need) Need {
	return func(s *E2ESuite) bool {
		for _, n := range needs {
			if n(s) {
				return true
			}
		}
		return false
	}
}

func And(needs ...Need) Need {
	return func(s *E2ESuite) bool {
		for _, n := range needs {
			if !n(s) {
				return false
			}
		}
		return true
	}
}
