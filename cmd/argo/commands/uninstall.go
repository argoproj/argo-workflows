package commands

import (
	"fmt"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	workflowclient "github.com/argoproj/argo/workflow/client"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	RootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().StringVar(&uninstallArgs.name, "name", common.DefaultControllerDeploymentName, "name of deployment")
	uninstallCmd.Flags().StringVar(&uninstallArgs.configMap, "configmap", common.DefaultConfigMapName(common.DefaultControllerDeploymentName), "name of configmap to uninstall")
	uninstallCmd.Flags().StringVar(&uninstallArgs.namespace, "install-namespace", common.DefaultControllerNamespace, "uninstall from a specific namespace")
}

type uninstallFlags struct {
	name      string // --name
	configMap string // --configmap
	namespace string // --install-namespace
}

var uninstallArgs uninstallFlags

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "uninstall controller and CRD",
	Run:   uninstall,
}

func uninstall(cmd *cobra.Command, args []string) {
	clientset = initKubeClient()

	// Delete the deployment
	deploymentsClient := clientset.AppsV1beta2().Deployments(uninstallArgs.namespace)
	deletePolicy := metav1.DeletePropagationForeground
	err := deploymentsClient.Delete(uninstallArgs.name, &metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	if err != nil {
		if !apierr.IsNotFound(err) {
			log.Fatalf("Failed to delete deployment '%s': %v", uninstallArgs.name, err)
		}
		fmt.Printf("Deployment '%s' in namespace '%s' not found\n", uninstallArgs.name, uninstallArgs.namespace)
	} else {
		fmt.Printf("Deployment '%s' deleted\n", uninstallArgs.name)
	}

	// Delete the configmap
	cmClient := clientset.CoreV1().ConfigMaps(uninstallArgs.namespace)
	err = cmClient.Delete(uninstallArgs.configMap, &metav1.DeleteOptions{})
	if err != nil {
		if !apierr.IsNotFound(err) {
			log.Fatalf("Failed to delete ConfigMap '%s': %v", uninstallArgs.configMap, err)
		}
		fmt.Printf("ConfigMap '%s' in namespace '%s' not found\n", uninstallArgs.configMap, uninstallArgs.namespace)
	} else {
		fmt.Printf("ConfigMap '%s' deleted\n", uninstallArgs.configMap)
	}

	// Delete the workflow CRD
	apiextensionsclientset, err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	err = workflowclient.DeleteCustomResourceDefinition(apiextensionsclientset)
	if err != nil {
		if !apierr.IsNotFound(err) {
			log.Fatalf("Failed to delete workflow CRD '%s': %v", wfv1.CRDFullName, err)
		}
		fmt.Printf("Workflow CRD '%s' not found\n", wfv1.CRDFullName)
	} else {
		fmt.Printf("Workflow CRD '%s' deleted\n", wfv1.CRDFullName)
	}
}
