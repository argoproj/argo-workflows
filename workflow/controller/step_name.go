package controller

import (
	"fmt"
	"strings"
)

func joinStepNodeName(jobName string, stepName string) string {
	return fmt.Sprintf("%s.%s", jobName, stepName)
}

func splitStepNodeName(n string) (string, string) {
	parts := strings.Split(n, ".")
	return parts[0], parts[1]
}
