package fixtures

import "os"

type Env struct {
}

func (s *Env) SetEnv() {
	_ = os.Setenv("ARGO_SERVER", "localhost:2746")
	_ = os.Setenv("ARGO_TOKEN_VERSION", "v2")
	_ = os.Setenv("ARGO_V2_TOKEN", "password")
}
func (s *Env) UnsetEnv() {
	_ = os.Unsetenv("ARGO_SERVER")
	_ = os.Unsetenv("ARGO_TOKEN_VERSION")
	_ = os.Unsetenv("ARGO_V2_TOKEN")
}
