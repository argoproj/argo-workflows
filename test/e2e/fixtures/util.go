package fixtures

import (
	"os/exec"

	log "github.com/sirupsen/logrus"
)

func runCli(diagnostics *Diagnostics, args []string) (string, error) {
	runArgs := append([]string{"-n", Namespace}, args...)
	output, err := exec.Command("../../dist/argo", runArgs...).CombinedOutput()
	stringOutput := string(output)
	diagnostics.Log(log.Fields{"args": args, "output": stringOutput, "err": err}, "Run CLI")
	return stringOutput, err
}
