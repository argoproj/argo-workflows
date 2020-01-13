package fixtures

import (
	"os/exec"
)

func runCli(args []string) (string, error) {
	runArgs := append([]string{"-n", Namespace}, args...)
	output, err := exec.Command("../../dist/argo", runArgs...).CombinedOutput()
	return string(output), err
}
