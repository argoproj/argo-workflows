package fixtures

import (
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/exec"
	"strings"
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


func GetServiceAccountToken() string {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{}
	config := clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)
	restConfig, err := config.ClientConfig()
	if err != nil {
		log.Fatal(err)
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Fatal(err)
	}
	secretList, err := clientset.CoreV1().Secrets("argo").List(metav1.ListOptions{})

	for _, sec := range secretList.Items {
		if strings.HasPrefix(sec.Name, "argo-server-token") {
			return string(sec.Data["token"])
		}
	}
	return ""
}