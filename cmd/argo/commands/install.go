package commands

import (
	"fmt"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	workflowclient "github.com/argoproj/argo/workflow/client"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/controller"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	apiv1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	RootCmd.AddCommand(installCmd)
	installCmd.Flags().StringVar(&installArgs.name, "name", common.DefaultControllerDeploymentName, "name of deployment")
	installCmd.Flags().StringVar(&installArgs.namespace, "install-namespace", common.DefaultControllerNamespace, "install into a specific namespace")
	installCmd.Flags().StringVar(&installArgs.configMap, "configmap", common.DefaultConfigMapName(common.DefaultControllerDeploymentName), "install controller using preconfigured configmap")
	installCmd.Flags().StringVar(&installArgs.controllerImage, "controller-image", common.DefaultControllerImage, "use a specified controller image")
	installCmd.Flags().StringVar(&installArgs.executorImage, "executor-image", common.DefaultExecutorImage, "use a specified executor image")
	installCmd.Flags().StringVar(&installArgs.serviceAccount, "service-account", "", "use a specified service account for the workflow-controller deployment")
}

type installFlags struct {
	name            string // --name
	namespace       string // --install-namespace
	configMap       string // --configmap
	controllerImage string // --controller-image
	executorImage   string // --executor-image
	serviceAccount  string // --service-account
}

var installArgs installFlags

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "install commands",
	Run:   install,
}

func install(cmd *cobra.Command, args []string) {
	fmt.Printf("Installing into namespace '%s'\n", installArgs.namespace)
	installCRD()
	installConfigMap()
	installController()
}

func installConfigMap() {
	clientset = initKubeClient()
	cmClient := clientset.CoreV1().ConfigMaps(installArgs.namespace)
	var wfConfig controller.WorkflowControllerConfig

	// install configmap if non-existant
	wfConfigMap, err := cmClient.Get(installArgs.configMap, metav1.GetOptions{})
	if err != nil {
		if !apierr.IsNotFound(err) {
			log.Fatalf("Failed lookup of ConfigMap '%s' in namespace '%s': %v", installArgs.configMap, installArgs.namespace, err)
		}
		// Create the config map
		wfConfig.ExecutorImage = installArgs.executorImage
		configBytes, err := yaml.Marshal(wfConfig)
		if err != nil {
			log.Fatalf("%+v", errors.InternalWrapError(err))
		}
		wfConfigMap.ObjectMeta.Name = installArgs.configMap
		wfConfigMap.Data = map[string]string{
			common.WorkflowControllerConfigMapKey: string(configBytes),
		}
		wfConfigMap, err = cmClient.Create(wfConfigMap)
		if err != nil {
			log.Fatalf("Failed to create ConfigMap '%s' in namespace '%s': %v", installArgs.configMap, installArgs.namespace, err)
		}
		fmt.Printf("ConfigMap '%s' created\n", installArgs.configMap)
	} else {
		fmt.Printf("ConfigMap '%s' already exists. Skip creation\n", installArgs.configMap)
	}
	configStr, ok := wfConfigMap.Data[common.WorkflowControllerConfigMapKey]
	if !ok {
		log.Fatalf("ConfigMap '%s' missing key '%s'", installArgs.configMap, common.WorkflowControllerConfigMapKey)
	}
	err = yaml.Unmarshal([]byte(configStr), &wfConfig)
	if err != nil {
		log.Fatalf("Failed to load controller configuration: %v", err)
	}
}

func installController() {
	clientset = initKubeClient()
	deploymentsClient := clientset.AppsV1beta2().Deployments(installArgs.namespace)
	controllerDeployment := appsv1beta2.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: installArgs.name,
		},
		Spec: appsv1beta2.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": installArgs.name,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": installArgs.name,
					},
				},
				Spec: apiv1.PodSpec{
					ServiceAccountName: installArgs.serviceAccount,
					Containers: []apiv1.Container{
						{
							Name:    installArgs.name,
							Image:   installArgs.controllerImage,
							Command: []string{"workflow-controller"},
							Args:    []string{"--configmap", installArgs.configMap},
							Env: []apiv1.EnvVar{
								apiv1.EnvVar{
									Name: common.EnvVarNamespace,
									ValueFrom: &apiv1.EnvVarSource{
										FieldRef: &apiv1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "metadata.namespace",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Create Deployment
	var result *appsv1beta2.Deployment
	var err error
	result, err = deploymentsClient.Create(&controllerDeployment)
	if err != nil {
		if !apierr.IsAlreadyExists(err) {
			log.Fatal(err)
		}
		result, err = deploymentsClient.Update(&controllerDeployment)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Existing deployment '%s' updated\n", result.GetObjectMeta().GetName())
	} else {
		fmt.Printf("Deployment '%s' created\n", result.GetObjectMeta().GetName())
	}
}

func installCRD() {
	clientset = initKubeClient()
	apiextensionsclientset, err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("Failed to create Workflow CRD: %v", err)
	}

	// initialize custom resource using a CustomResourceDefinition if it does not exist
	result, err := workflowclient.CreateCustomResourceDefinition(apiextensionsclientset)
	if err != nil {
		if !apierr.IsAlreadyExists(err) {
			log.Fatalf("Failed to create Workflow CRD: %v", err)
		}
		fmt.Printf("Workflow CRD '%s' already exists\n", wfv1.CRDFullName)
	} else {
		fmt.Printf("Workflow CRD '%s' created\n", result.GetObjectMeta().GetName())
	}
}
