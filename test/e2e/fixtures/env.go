package fixtures

import "os"

type Env struct {
}

func (s *Env) SetEnv(token string) {
	_ = os.Setenv("ARGO_SERVER", "localhost:2746")
	_ = os.Setenv("ARGO_TOKEN", token)
}
func (s *Env) UnsetEnv() {
	_ = os.Unsetenv("ARGO_SERVER")
	_ = os.Unsetenv("ARGO_TOKEN")
}
