package fixtures

import (
	"os"

	"k8s.io/client-go/rest"
)

type Env struct {
}

func (s *Env) SetEnv(restConfig *rest.Config) {
	_ = os.Setenv("ARGO_SERVER", "localhost:2746")
	_ = os.Setenv("ARGO_TOKEN", GetServiceAccountToken(restConfig))
}
func (s *Env) UnsetEnv() {
	_ = os.Unsetenv("ARGO_SERVER")
	_ = os.Unsetenv("ARGO_TOKEN")
}
