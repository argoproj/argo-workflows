package fixtures

import (
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func runCli(diagnostics *Diagnostics, args []string) (string, error) {
	runArgs := append([]string{"-n", Namespace}, args...)
	cmd := exec.Command("../../dist/argo", runArgs...)
	cmd.Env = os.Environ()
	output, err := exec.Command("../../dist/argo", runArgs...).CombinedOutput()
	stringOutput := string(output)
	diagnostics.Log(log.Fields{"args": args, "output": stringOutput, "err": err}, "Run CLI")
	return stringOutput, err
}

func GetServiceAccountToken(restConfig *rest.Config) string {
	// create the clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Fatal(err)
	}
	secretList, err := clientset.CoreV1().Secrets("argo").List(metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	for _, sec := range secretList.Items {
		if strings.HasPrefix(sec.Name, "argo-server-token") {
			return string(sec.Data["token"])
		}
	}
	return ""
}