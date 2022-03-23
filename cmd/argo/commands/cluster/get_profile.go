package cluster

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func newGetProfileCommand() *cobra.Command {
	var (
		server                   string
		certificateAuthorityFile string
		insecureSkipTLSVerify    bool
		context                  string
	)
	cmd := &cobra.Command{
		Use:   "get-profile cluster namespace service_account_name",
		Short: "print the profile for the  cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if len(args) != 3 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			cluster, namespace, serviceAccountName := args[0], args[1], args[2]

			if context == "" {
				context = cluster
			}

			clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				clientcmd.NewDefaultClientConfigLoadingRules(),
				&clientcmd.ConfigOverrides{CurrentContext: context},
			)

			config, err := clientConfig.ClientConfig()
			if err != nil {
				return err
			}

			if server == "" {
				server = config.Host
			}

			client := kubernetes.NewForConfigOrDie(config)

			serviceAccount, err := client.CoreV1().ServiceAccounts(namespace).Get(ctx, serviceAccountName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			secretName := serviceAccount.Secrets[0].Name

			secret, err := client.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			ca := secret.Data[apiv1.ServiceAccountRootCAKey]
			token := secret.Data[apiv1.ServiceAccountTokenKey]

			if certificateAuthorityFile != "" {
				ca, err = os.ReadFile(certificateAuthorityFile)
				if err != nil {
					return err
				}
			}

			if insecureSkipTLSVerify {
				println("⚠️ do not use insecure skip verify in production")
				// specifying a root certificates file with the insecure flag is not allowed
				ca = nil
			}

			kubeconfig := api.Config{
				Kind:       "Config",
				APIVersion: "v1",
				Clusters: map[string]*api.Cluster{
					cluster: {Server: server, CertificateAuthorityData: ca, InsecureSkipTLSVerify: insecureSkipTLSVerify},
				},
				AuthInfos: map[string]*api.AuthInfo{
					serviceAccountName: {Token: string(token)},
				},
				Contexts: map[string]*api.Context{
					cluster: {Cluster: cluster, AuthInfo: serviceAccountName},
				},
				CurrentContext: cluster,
			}

			data, err := clientcmd.Write(kubeconfig)
			if err != nil {
				return err
			}

			profile := &apiv1.Secret{
				TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("argo.%s", cluster),
					Labels: map[string]string{
						common.LabelKeyCluster: cluster,
					},
				},
				Data: map[string][]byte{"kubeconfig": data},
			}

			data, err = yaml.Marshal(profile)
			if err != nil {
				return err
			}

			_, _ = os.Stdout.WriteString("# This is an auto-generated file. DO NOT EDIT\n")
			_, _ = os.Stdout.Write(data)

			return nil
		},
	}
	cmd.Flags().StringVar(&server, "server", "", "URL for  server")
	cmd.Flags().StringVar(&certificateAuthorityFile, "certificate-authority-file", "", "file containing  certificate authority")
	cmd.Flags().BoolVar(&insecureSkipTLSVerify, "insecure-skip-tls-verify", false, "skip certificate for  server, do not use in production")
	cmd.Flags().StringVar(&context, "context", "", " context")
	return cmd
}
