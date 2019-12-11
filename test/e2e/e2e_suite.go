package e2e

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/stretchr/testify/suite"
	// load the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/argoproj/argo/cmd/argo/commands"
)

var kubeConfig = flag.String("kubeconfig", filepath.Join(os.Getenv("HOME"), ".kube", "config"), "Path to Kubernetes config file")

func init() {
	_ = commands.NewCommand()
}

type E2ESuite struct {
	suite.Suite
}

func (suite *E2ESuite) SetupSuite() {
	_, err := os.Stat(*kubeConfig)
	if os.IsNotExist(err) {
		suite.T().Skip("Skipping test: " + err.Error())
		return
	}
}

func (suite *E2ESuite) Given() *Given {
	return &Given{suite: suite}
}
