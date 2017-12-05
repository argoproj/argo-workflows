package commands

import (
	"fmt"

	wfv1 "github.com/argoproj/argo/api/workflow/v1alpha1"
	"github.com/argoproj/argo/errors"
	workflowclient "github.com/argoproj/argo/workflow/client"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/controller"
	"github.com/ghodss/yaml"
	goversion "github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	apiv1 "k8s.io/api/core/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const clusterAdmin = "cluster-admin"

func init() {
	RootCmd.AddCommand(installCmd)
	installCmd.Flags().StringVar(&installArgs.controllerName, "controller-name", common.DefaultControllerDeploymentName, "name of controller deployment")
	installCmd.Flags().StringVar(&installArgs.uiName, "ui-name", common.DefaultUiDeploymentName, "name of ui deployment")
	installCmd.Flags().StringVar(&installArgs.namespace, "install-namespace", common.DefaultControllerNamespace, "install into a specific namespace")
	installCmd.Flags().StringVar(&installArgs.configMap, "configmap", common.DefaultConfigMapName(common.DefaultControllerDeploymentName), "install controller using preconfigured configmap")
	installCmd.Flags().StringVar(&installArgs.controllerImage, "controller-image", common.DefaultControllerImage, "use a specified controller image")
	installCmd.Flags().StringVar(&installArgs.uiImage, "ui-image", common.DefaultUiImage, "use a specified ui image")
	installCmd.Flags().StringVar(&installArgs.executorImage, "executor-image", common.DefaultExecutorImage, "use a specified executor image")
	installCmd.Flags().StringVar(&installArgs.serviceAccount, "service-account", "", "use a specified service account for the workflow-controller deployment")
}

type installFlags struct {
	controllerName  string // --name
	uiName          string // --ui-name
	namespace       string // --install-namespace
	configMap       string // --configmap
	controllerImage string // --controller-image
	uiImage         string // --ui-image
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
	clientset = initKubeClient()
	kubernetesVersionCheck(clientset)
	installCRD(clientset)
	if installArgs.serviceAccount == "" {
		if clusterAdminExists(clientset) {
			seviceAccountName := ArgoServiceAccount
			createServiceAccount(clientset, seviceAccountName)
			createClusterRoleBinding(clientset, seviceAccountName)
			installArgs.serviceAccount = seviceAccountName
		}
	}
	installConfigMap(clientset)
	installController(clientset)
	installUi(clientset)
}

func clusterAdminExists(clientset *kubernetes.Clientset) bool {
	clusterRoles := clientset.RbacV1beta1().ClusterRoles()
	_, err := clusterRoles.Get(clusterAdmin, metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) {
			return false
		}
		log.Fatalf("Failed to lookup 'cluster-admin' role: %v", err)
	}
	return true
}

func createServiceAccount(clientset *kubernetes.Clientset, serviceAccountName string) {
	serviceAccount := apiv1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: installArgs.namespace,
		},
	}
	_, err := clientset.CoreV1().ServiceAccounts(installArgs.namespace).Create(&serviceAccount)
	if err != nil {
		if !apierr.IsAlreadyExists(err) {
			log.Fatalf("Failed to create service account '%s': %v\n", serviceAccountName, err)
		}
		fmt.Printf("ServiceAccount '%s' already exists\n", serviceAccountName)
		return
	}
	fmt.Printf("ServiceAccount '%s' created\n", serviceAccountName)
}

func createClusterRoleBinding(clientset *kubernetes.Clientset, serviceAccountName string) {
	roleBinding := rbacv1beta1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1beta1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ArgoClusterRole,
		},
		RoleRef: rbacv1beta1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterAdmin,
		},
		Subjects: []rbacv1beta1.Subject{
			{
				Kind:      rbacv1beta1.ServiceAccountKind,
				Name:      serviceAccountName,
				Namespace: installArgs.namespace,
			},
		},
	}

	_, err := clientset.RbacV1beta1().ClusterRoleBindings().Create(&roleBinding)
	if err != nil {
		if !apierr.IsAlreadyExists(err) {
			log.Fatalf("Failed to create ClusterRoleBinding %s: %v\n", ArgoClusterRole, err)
		}
		fmt.Printf("ClusterRoleBinding '%s' already exists\n", ArgoClusterRole)
		return
	}
	fmt.Printf("ClusterRoleBinding '%s' created, bound '%s' to '%s'\n", ArgoClusterRole, serviceAccountName, clusterAdmin)
}

func kubernetesVersionCheck(clientset *kubernetes.Clientset) {
	// Check if the Kubernetes version is >= 1.8
	versionInfo, err := clientset.ServerVersion()
	if err != nil {
		log.Fatalf("Failed to get Kubernetes version: %v", err)
	}

	serverVersion, err := goversion.NewVersion(versionInfo.String())
	if err != nil {
		log.Fatalf("Failed to create version: %v", err)
	}

	minVersion, err := goversion.NewVersion("1.8")
	if err != nil {
		log.Fatalf("Failed to create minimum version: %v", err)
	}

	if serverVersion.LessThan(minVersion) {
		log.Fatalf("Server version %v < %v. Installation won't proceed...\n", serverVersion, minVersion)
	}

	fmt.Printf("Proceeding with Kubernetes version %v\n", serverVersion)
}

func installConfigMap(clientset *kubernetes.Clientset) {
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
		fmt.Printf("ConfigMap '%s' already exists\n", installArgs.configMap)
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

func installController(clientset *kubernetes.Clientset) {
	if installArgs.serviceAccount == "" {
		fmt.Printf("Using default service account for '%s' deployment\n", installArgs.controllerName)
	} else {
		fmt.Printf("Using service account '%s' for '%s' deployment\n", installArgs.serviceAccount, installArgs.controllerName)
	}

	deploymentsClient := clientset.AppsV1beta2().Deployments(installArgs.namespace)
	controllerDeployment := appsv1beta2.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: installArgs.controllerName,
		},
		Spec: appsv1beta2.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": installArgs.controllerName,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": installArgs.controllerName,
					},
				},
				Spec: apiv1.PodSpec{
					ServiceAccountName: installArgs.serviceAccount,
					Containers: []apiv1.Container{
						{
							Name:    installArgs.controllerName,
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

func installUi(clientset *kubernetes.Clientset) {
	if installArgs.serviceAccount == "" {
		fmt.Printf("Using default service account for '%s' deployment\n", installArgs.controllerName)
	} else {
		fmt.Printf("Using service account '%s' for '%s' deployment\n", installArgs.serviceAccount, installArgs.controllerName)
	}

	deploymentsClient := clientset.AppsV1beta2().Deployments(installArgs.namespace)
	uiDeployment := appsv1beta2.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: installArgs.uiName,
		},
		Spec: appsv1beta2.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": installArgs.uiName,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": installArgs.uiName,
					},
				},
				Spec: apiv1.PodSpec{
					ServiceAccountName: installArgs.serviceAccount,
					Containers: []apiv1.Container{
						{
							Name:  installArgs.uiName,
							Image: installArgs.uiImage,
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
								apiv1.EnvVar{
									Name:  "IN_CLUSTER",
									Value: "true",
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
	result, err = deploymentsClient.Create(&uiDeployment)
	if err != nil {
		if !apierr.IsAlreadyExists(err) {
			log.Fatal(err)
		}
		result, err = deploymentsClient.Update(&uiDeployment)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Existing deployment '%s' updated\n", result.GetObjectMeta().GetName())
	} else {
		fmt.Printf("Deployment '%s' created\n", result.GetObjectMeta().GetName())
	}
}

func installCRD(clientset *kubernetes.Clientset) {
	apiextensionsclientset, err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("Failed to create CustomResourceDefinition '%s': %v", wfv1.CRDFullName, err)
	}

	// initialize custom resource using a CustomResourceDefinition if it does not exist
	result, err := workflowclient.CreateCustomResourceDefinition(apiextensionsclientset)
	if err != nil {
		if !apierr.IsAlreadyExists(err) {
			log.Fatalf("Failed to create CustomResourceDefinition: %v", err)
		}
		fmt.Printf("CustomResourceDefinition '%s' already exists\n", wfv1.CRDFullName)
	} else {
		fmt.Printf("CustomResourceDefinition '%s' created\n", result.GetObjectMeta().GetName())
	}
}
