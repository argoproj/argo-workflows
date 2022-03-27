package fixtures

import (
	"os"
)

type Need func(s *E2ESuite) (met bool, message string)

var (
	CI Need = func(s *E2ESuite) (bool, string) {
		return os.Getenv("CI") != "", "CI"
	}
	Emissary = Executor("emissary")
	PNS      = Executor("pns")
)

func Executor(e string) Need {
	return func(s *E2ESuite) (bool, string) {
		v := s.Config.ContainerRuntimeExecutor
		if v == "" {
			v = "emissary"
		}
		return v == e, e
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
