package commands

import (
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func NewDescribeCommand() *cobra.Command {

	var (
		printer DescribePrinter
	)
	var command = &cobra.Command{
		Use:   "describe POD",
		Short: "view details of a pod",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			conf, err := clientConfig.ClientConfig()
			errors.CheckError(err)
			printer.kubeClient = kubernetes.NewForConfigOrDie(conf)
			namespace, _, err := clientConfig.Namespace()
			errors.CheckError(err)
			buf, err := printer.DescribePod(args[0], namespace)
			errors.CheckError(err)
			_, err := os.Stdout.Write(buf[:])
			errors.CheckError(err)

		}, // --- end of func ---
	}
	return command
}

type DescribePrinter struct {
	kubeClient kubernetes.Interface
}

func (p *DescribePrinter) DescribePod(podName, podNamespace string) ([]byte, error) {
	pod, err := p.kubeClient.CoreV1().Pods(podNamespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	buf, err := yaml.Marshal(pod)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
