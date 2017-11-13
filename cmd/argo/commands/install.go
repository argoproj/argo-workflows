package commands

import (
	"fmt"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/controller"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	RootCmd.AddCommand(installCmd)
	installCmd.Flags().StringVar(&installArgs.name, "name", "workflow-controller", "name of deployment")
	installCmd.Flags().StringVar(&installArgs.namespace, "namespace", apiv1.NamespaceDefault, "install into a specific namespace")
	installCmd.Flags().StringVar(&installArgs.configMap, "configmap", common.DefaultWorkflowControllerConfigMap, "install controller using preconfigured configmap")
	installCmd.Flags().StringVar(&installArgs.controllerImage, "controller-image", common.DefaultControllerImage, "use a specified controller image")
	installCmd.Flags().StringVar(&installArgs.executorImage, "executor-image", common.DefaultExecutorImage, "use a specified executor image")
}

type installFlags struct {
	name            string // --name
	namespace       string // --namespace
	configMap       string // --configmap
	controllerImage string // --controller-image
	executorImage   string // --executor-image
}

var installArgs installFlags

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "install commands",
	Run:   install,
}

func install(cmd *cobra.Command, args []string) {
	installConfigMap()
	installController()
}

func installConfigMap() {
	clientset = initKubeClient()

	cmClient := clientset.CoreV1().ConfigMaps(installArgs.namespace)
	var err error
	var wfConfigMap *apiv1.ConfigMap
	var wfConfig controller.WorkflowControllerConfig

	// install configmap if non-existant
	wfConfigMap, err = cmClient.Get(installArgs.configMap, metav1.GetOptions{})
	if err != nil {
		if !apierr.IsNotFound(err) {
			log.Fatalf("Failed lookup of configmap '%s' in namespace '%s': %v", installArgs.configMap, installArgs.namespace, err)
		}
		// Create the config map
		fmt.Printf("Creating '%s' config map in '%s'\n", installArgs.configMap, installArgs.namespace)
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
			log.Fatalf("Failed to create configmap '%s' in namespace '%s': %v", installArgs.configMap, installArgs.namespace, err)
		}
		fmt.Printf("Config map '%s' created\n", installArgs.configMap)
	} else {
		fmt.Printf("Found existing configmap '%s' in namespace '%s'. Skip configmap creation\n", installArgs.configMap, installArgs.namespace)
	}
	configStr, ok := wfConfigMap.Data[common.WorkflowControllerConfigMapKey]
	if !ok {
		log.Fatalf("Configmap '%s' missing key '%s'", installArgs.configMap, common.WorkflowControllerConfigMapKey)
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
					Containers: []apiv1.Container{
						{
							Name:    installArgs.name,
							Image:   installArgs.controllerImage,
							Command: []string{"workflow-controller"},
							Args:    []string{"--configmap", installArgs.configMap},
						},
					},
				},
			},
		},
	}

	// Create Deployment
	fmt.Printf("Creating '%s' deployment...\n", controllerDeployment.ObjectMeta.Name)
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
		fmt.Printf("Successfully updated existing '%s' deployment.\n", result.GetObjectMeta().GetName())
	} else {
		fmt.Printf("Successfully created '%s' deployment.\n", result.GetObjectMeta().GetName())
	}
}
