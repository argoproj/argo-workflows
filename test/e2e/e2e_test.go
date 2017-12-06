package e2e

import (
	"flag"
)

var kubeConfig = flag.String("kubeconfig", "", "Path to Kubernetes config file")
