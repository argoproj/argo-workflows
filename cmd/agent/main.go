package main

import (
	"github.com/argoproj/pkg/cli"
	"github.com/argoproj/pkg/errors"
	kubecli "github.com/argoproj/pkg/kube/cli"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/agent"
)

func main() {
	var (
		clientConfig clientcmd.ClientConfig
		secure       bool
		logLevel     string
	)
	cmd := cobra.Command{
		PreRun: func(cmd *cobra.Command, args []string) {
			cli.SetLogLevel(logLevel)
		},
		Run: func(cmd *cobra.Command, args []string) {
			r, err := clientConfig.ClientConfig()
			errors.CheckError(err)
			clientset, err := kubernetes.NewForConfig(r)
			errors.CheckError(err)
			namespace, _, err := clientConfig.Namespace()
			errors.CheckError(err)
			a, err := agent.NewAgent(clientset, namespace, secure)
			errors.CheckError(err)
			err = a.Run()
			errors.CheckError(err)
		}}
	clientConfig = kubecli.AddKubectlFlagsToCmd(&cmd)
	cmd.PersistentFlags().StringVar(&logLevel, "loglevel", "info", "Set the logging level. One of: debug|info|warn|error")
	cmd.PersistentFlags().BoolVarP(&secure, "secure", "e", true, "Serve using TLS")
	err := cmd.Execute()
	errors.CheckError(err)
}
