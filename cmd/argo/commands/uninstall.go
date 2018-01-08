package commands

import (
	"fmt"

	"github.com/argoproj/argo/pkg/apis/workflow"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	RootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().StringVar(&uninstallArgs.controllerName, "controller-name", common.DefaultControllerDeploymentName, "name of controller deployment")
	uninstallCmd.Flags().StringVar(&uninstallArgs.uiName, "ui-name", common.DefaultUiDeploymentName, "name of ui deployment")
	uninstallCmd.Flags().StringVar(&uninstallArgs.configMap, "configmap", common.DefaultConfigMapName(common.DefaultControllerDeploymentName), "name of configmap to uninstall")
	uninstallCmd.Flags().StringVar(&uninstallArgs.namespace, "install-namespace", common.DefaultControllerNamespace, "uninstall from a specific namespace")
}

type uninstallFlags struct {
	controllerName string // --controller-name
	uiName         string // --ui-name
	configMap      string // --configmap
	namespace      string // --install-namespace
}

var uninstallArgs uninstallFlags

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "uninstall Argo",
	Run:   uninstall,
}

func uninstall(cmd *cobra.Command, args []string) {
	clientset = initKubeClient()
	fmt.Printf("Uninstalling from namespace '%s'\n", uninstallArgs.namespace)
	// Delete the UI service
	svcClient := clientset.CoreV1().Services(uninstallArgs.namespace)
	err := svcClient.Delete(ArgoServiceName, &metav1.DeleteOptions{})
	if err != nil {
		if !apierr.IsNotFound(err) {
			log.Fatalf("Failed to delete service '%s': %v", ArgoServiceName, err)
		}
		fmt.Printf("Service '%s' in namespace '%s' not found\n", ArgoServiceName, uninstallArgs.namespace)
	} else {
		fmt.Printf("Service '%s' deleted\n", ArgoServiceName)
	}

	// Delete the UI and workflow-controller deployment
	deploymentsClient := clientset.AppsV1beta2().Deployments(uninstallArgs.namespace)
	deletePolicy := metav1.DeletePropagationForeground
	for _, depName := range []string{uninstallArgs.uiName, uninstallArgs.controllerName} {
		err := deploymentsClient.Delete(depName, &metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
		if err != nil {
			if !apierr.IsNotFound(err) {
				log.Fatalf("Failed to delete deployment '%s': %v", depName, err)
			}
			fmt.Printf("Deployment '%s' in namespace '%s' not found\n", depName, uninstallArgs.namespace)
		} else {
			fmt.Printf("Deployment '%s' deleted\n", depName)
		}
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
	err = common.DeleteCustomResourceDefinition(apiextensionsclientset)
	if err != nil {
		if !apierr.IsNotFound(err) {
			log.Fatalf("Failed to delete CustomResourceDefinition '%s': %v", workflow.FullName, err)
		}
		fmt.Printf("CustomResourceDefinition '%s' not found\n", workflow.FullName)
	} else {
		fmt.Printf("CustomResourceDefinition '%s' deleted\n", workflow.FullName)
	}

	// Delete role binding
	if err := clientset.RbacV1beta1().ClusterRoleBindings().Delete(ArgoClusterRole, &metav1.DeleteOptions{}); err != nil {
		if !apierr.IsNotFound(err) {
			log.Fatalf("Failed to check clusterRoleBinding: %v\n", err)
		}
		fmt.Printf("ClusterRoleBinding '%s' not found\n", ArgoClusterRole)
	} else {
		fmt.Printf("ClusterRoleBinding '%s' deleted\n", ArgoClusterRole)
	}

	// Delete service account
	if err := clientset.CoreV1().ServiceAccounts(uninstallArgs.namespace).Delete(ArgoServiceAccount, &metav1.DeleteOptions{}); err != nil {
		if !apierr.IsNotFound(err) {
			log.Fatalf("Failed to get service accounts: %v\n", err)
		}
		fmt.Printf("ServiceAccount '%s' in namespace '%s' not found\n", ArgoServiceAccount, uninstallArgs.namespace)
	} else {
		fmt.Printf("ServiceAccount '%s' deleted\n", ArgoServiceAccount)
	}
}
