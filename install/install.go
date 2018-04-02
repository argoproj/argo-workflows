package install

import (
	"fmt"
	"strconv"

	"github.com/argoproj/argo"
	"github.com/argoproj/argo-cd/util/diff"
	"github.com/argoproj/argo-cd/util/kube"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/controller"
	"github.com/ghodss/yaml"
	"github.com/gobuffalo/packr"
	goversion "github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"github.com/yudai/gojsondiff/formatter"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type InstallOptions struct {
	Upgrade          bool   // --upgrade
	DryRun           bool   // --dry-run
	Namespace        string // --namespace
	InstanceID       string // --instanceid
	ConfigMap        string // --configmap
	ControllerImage  string // --controller-image
	ServiceAccount   string // --service-account
	ExecutorImage    string // --executor-image
	UIImage          string // --ui-image
	UIBaseHref       string // --ui-base-href
	UIServiceAccount string // --ui-service-account
	EnableWebConsole bool   // --enable-web-console
	ImagePullPolicy  string // --image-pull-policy
}

type Installer struct {
	InstallOptions
	box           packr.Box
	config        *rest.Config
	dynClientPool dynamic.ClientPool
	disco         discovery.DiscoveryInterface
	rbacSupported *bool
	clientset     *kubernetes.Clientset
}

func NewInstaller(config *rest.Config, opts InstallOptions) (*Installer, error) {
	shallowCopy := *config
	inst := Installer{
		InstallOptions: opts,
		box:            packr.NewBox("./manifests"),
		config:         &shallowCopy,
	}
	var err error
	inst.dynClientPool = dynamic.NewDynamicClientPool(inst.config)
	inst.disco, err = discovery.NewDiscoveryClientForConfig(inst.config)
	if err != nil {
		return nil, err
	}
	inst.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// Install installs the Argo controller and UI in the given Namespace
func (i *Installer) Install() {
	if !i.DryRun {
		fmt.Printf("Installing Argo %s into namespace '%s'\n", argo.GetVersion(), i.Namespace)
		kubernetesVersionCheck(i.clientset)
	}
	i.InstallWorkflowCRD()
	i.InstallWorkflowController()
	i.InstallArgoUI()
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

// IsRBACSupported returns whether or not RBAC is supported on the cluster
func (i *Installer) IsRBACSupported() bool {
	if i.rbacSupported != nil {
		return *i.rbacSupported
	}
	// TODO: figure out the proper way to test if RBAC is enabled
	clusterRoles := i.clientset.RbacV1().ClusterRoles()
	_, err := clusterRoles.Get("cluster-admin", metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) {
			f := false
			i.rbacSupported = &f
			return false
		}
		log.Fatalf("Failed to lookup 'cluster-admin' role: %v", err)
	}
	t := true
	i.rbacSupported = &t
	return true

}

func (i *Installer) InstallWorkflowCRD() {
	var workflowCRD apiextensionsv1beta1.CustomResourceDefinition
	i.unmarshalManifest("01_workflow-crd.yaml", &workflowCRD)
	obj := kube.MustToUnstructured(&workflowCRD)
	i.MustInstallResource(obj)
}

func (i *Installer) InstallWorkflowController() {
	var workflowControllerServiceAccount apiv1.ServiceAccount
	var workflowControllerClusterRole rbacv1.ClusterRole
	var workflowControllerClusterRoleBinding rbacv1.ClusterRoleBinding
	//var workflowControllerConfigMap apiv1.ConfigMap
	var workflowControllerDeployment appsv1beta2.Deployment
	i.unmarshalManifest("02a_workflow-controller-sa.yaml", &workflowControllerServiceAccount)
	i.unmarshalManifest("02b_workflow-controller-cluster-role.yaml", &workflowControllerClusterRole)
	i.unmarshalManifest("02c_workflow-controller-cluster-rolebinding.yaml", &workflowControllerClusterRoleBinding)
	//i.unmarshalManifest("02d_workflow-controller-configmap.yaml", &workflowControllerConfigMap)
	i.unmarshalManifest("02e_workflow-controller-deployment.yaml", &workflowControllerDeployment)
	workflowControllerDeployment.Spec.Template.Spec.Containers[0].Image = i.ControllerImage
	workflowControllerDeployment.Spec.Template.Spec.Containers[0].ImagePullPolicy = apiv1.PullPolicy(i.ImagePullPolicy)
	if i.ServiceAccount == "" {
		i.MustInstallResource(kube.MustToUnstructured(&workflowControllerServiceAccount))
		if i.IsRBACSupported() {
			workflowControllerClusterRoleBinding.Subjects[0].Namespace = i.Namespace
			i.MustInstallResource(kube.MustToUnstructured(&workflowControllerClusterRole))
			i.MustInstallResource(kube.MustToUnstructured(&workflowControllerClusterRoleBinding))
		}
	} else {
		workflowControllerDeployment.Spec.Template.Spec.ServiceAccountName = i.ServiceAccount
	}
	//i.MustInstallResource(kube.MustToUnstructured(&workflowControllerConfigMap))
	i.installConfigMap(i.clientset)
	i.MustInstallResource(kube.MustToUnstructured(&workflowControllerDeployment))
}

func (i *Installer) InstallArgoUI() {
	var argoUIServiceAccount apiv1.ServiceAccount
	var argoUIClusterRole rbacv1.ClusterRole
	var argoUIClusterRoleBinding rbacv1.ClusterRoleBinding
	var argoUIDeployment appsv1beta2.Deployment
	var argoUIService apiv1.Service
	i.unmarshalManifest("03a_argo-ui-sa.yaml", &argoUIServiceAccount)
	i.unmarshalManifest("03b_argo-ui-cluster-role.yaml", &argoUIClusterRole)
	i.unmarshalManifest("03c_argo-ui-cluster-rolebinding.yaml", &argoUIClusterRoleBinding)
	i.unmarshalManifest("03d_argo-ui-deployment.yaml", &argoUIDeployment)
	i.unmarshalManifest("03e_argo-ui-service.yaml", &argoUIService)
	argoUIDeployment.Spec.Template.Spec.Containers[0].Image = i.UIImage
	argoUIDeployment.Spec.Template.Spec.Containers[0].ImagePullPolicy = apiv1.PullPolicy(i.ImagePullPolicy)
	setEnv(&argoUIDeployment, "ENABLE_WEB_CONSOLE", strconv.FormatBool(i.EnableWebConsole))
	setEnv(&argoUIDeployment, "BASE_HREF", i.UIBaseHref)
	if i.UIServiceAccount == "" {
		i.MustInstallResource(kube.MustToUnstructured(&argoUIServiceAccount))
		if i.IsRBACSupported() {
			argoUIClusterRoleBinding.Subjects[0].Namespace = i.Namespace
			i.MustInstallResource(kube.MustToUnstructured(&argoUIClusterRole))
			i.MustInstallResource(kube.MustToUnstructured(&argoUIClusterRoleBinding))
		}
	} else {
		argoUIDeployment.Spec.Template.Spec.ServiceAccountName = i.UIServiceAccount
	}
	i.MustInstallResource(kube.MustToUnstructured(&argoUIDeployment))
	i.MustInstallResource(kube.MustToUnstructured(&argoUIService))
}

func setEnv(dep *appsv1beta2.Deployment, key, val string) {
	ctr := dep.Spec.Template.Spec.Containers[0]
	for i, env := range ctr.Env {
		if env.Name == key {
			env.Value = val
			ctr.Env[i] = env
			return
		}
	}
	ctr.Env = append(ctr.Env, apiv1.EnvVar{Name: key, Value: val})
}

func (i *Installer) unmarshalManifest(fileName string, obj interface{}) {
	yamlBytes, err := i.box.MustBytes(fileName)
	checkError(err)
	err = yaml.Unmarshal(yamlBytes, obj)
	checkError(err)
}

func (i *Installer) MustInstallResource(obj *unstructured.Unstructured) *unstructured.Unstructured {
	obj, err := i.InstallResource(obj)
	checkError(err)
	return obj
}

func isNamespaced(obj *unstructured.Unstructured) bool {
	switch obj.GetKind() {
	case "Namespace", "ClusterRole", "ClusterRoleBinding", "CustomResourceDefinition":
		return false
	}
	return true
}

// InstallResource creates or updates a resource. If installed resource is up-to-date, does nothing
func (i *Installer) InstallResource(obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if isNamespaced(obj) {
		obj.SetNamespace(i.Namespace)
	}
	// remove 'creationTimestamp' and 'status' fields from object so that the diff will not be modified
	obj.SetCreationTimestamp(metav1.Time{})
	delete(obj.Object, "status")
	if i.DryRun {
		printYAML(obj)
		return nil, nil
	}
	gvk := obj.GroupVersionKind()
	dclient, err := i.dynClientPool.ClientForGroupVersionKind(gvk)
	if err != nil {
		return nil, err
	}
	apiResource, err := kube.ServerResourceForGroupVersionKind(i.disco, gvk)
	if err != nil {
		return nil, err
	}
	reIf := dclient.Resource(apiResource, i.Namespace)
	liveObj, err := reIf.Create(obj)
	if err == nil {
		fmt.Printf("%s '%s' created\n", liveObj.GetKind(), liveObj.GetName())
		return liveObj, nil
	}
	if !apierr.IsAlreadyExists(err) {
		return nil, err
	}
	liveObj, err = reIf.Get(obj.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	diffRes := diff.Diff(obj, liveObj)
	if !diffRes.Modified {
		fmt.Printf("%s '%s' up-to-date\n", liveObj.GetKind(), liveObj.GetName())
		return liveObj, nil
	}
	if !i.Upgrade {
		log.Println(diffRes.ASCIIFormat(obj, formatter.AsciiFormatterConfig{}))
		return nil, fmt.Errorf("%s '%s' already exists. Rerun with --upgrade to update", obj.GetKind(), obj.GetName())
	}
	liveObj, err = reIf.Update(obj)
	if err != nil {
		return nil, err
	}
	fmt.Printf("%s '%s' updated\n", liveObj.GetKind(), liveObj.GetName())
	return liveObj, nil
}

func printYAML(obj interface{}) {
	objBytes, err := yaml.Marshal(obj)
	if err != nil {
		log.Fatalf("Failed to marshal %v", obj)
	}
	fmt.Printf("---\n%s\n", string(objBytes))
}

// checkError is a convenience function to exit if an error is non-nil and exit if it was
func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func (i *Installer) installConfigMap(clientset *kubernetes.Clientset) {
	cmClient := clientset.CoreV1().ConfigMaps(i.Namespace)
	wfConfig := controller.WorkflowControllerConfig{
		ExecutorImage: i.ExecutorImage,
		InstanceID:    i.InstanceID,
	}
	configBytes, err := yaml.Marshal(wfConfig)
	if err != nil {
		log.Fatalf("%+v", errors.InternalWrapError(err))
	}
	wfConfigMap := apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      i.ConfigMap,
			Namespace: i.Namespace,
		},
		Data: map[string]string{
			common.WorkflowControllerConfigMapKey: string(configBytes),
		},
	}
	if i.DryRun {
		printYAML(wfConfigMap)
		return
	}
	_, err = cmClient.Create(&wfConfigMap)
	if err != nil {
		if !apierr.IsAlreadyExists(err) {
			log.Fatalf("Failed to create ConfigMap '%s' in namespace '%s': %v", i.ConfigMap, i.Namespace, err)
		}
		// Configmap already exists. Check if existing configmap needs an update to a new executor image
		existingCM, err := cmClient.Get(i.ConfigMap, metav1.GetOptions{})
		if err != nil {
			log.Fatalf("Failed to retrieve ConfigMap '%s' in namespace '%s': %v", i.ConfigMap, i.Namespace, err)
		}
		configStr, ok := existingCM.Data[common.WorkflowControllerConfigMapKey]
		if !ok {
			log.Fatalf("ConfigMap '%s' missing key '%s'", i.ConfigMap, common.WorkflowControllerConfigMapKey)
		}
		var existingConfig controller.WorkflowControllerConfig
		err = yaml.Unmarshal([]byte(configStr), &existingConfig)
		if err != nil {
			log.Fatalf("Failed to load controller configuration: %v", err)
		}
		if existingConfig.ExecutorImage == wfConfig.ExecutorImage {
			fmt.Printf("Existing ConfigMap '%s' up-to-date\n", i.ConfigMap)
			return
		}
		if !i.Upgrade {
			log.Fatalf("ConfigMap '%s' requires upgrade. Rerun with --upgrade to update the configuration", i.ConfigMap)
		}
		existingConfig.ExecutorImage = i.ExecutorImage
		configBytes, err := yaml.Marshal(existingConfig)
		if err != nil {
			log.Fatalf("%+v", errors.InternalWrapError(err))
		}
		existingCM.Data = map[string]string{
			common.WorkflowControllerConfigMapKey: string(configBytes),
		}
		_, err = cmClient.Update(existingCM)
		if err != nil {
			log.Fatalf("Failed to update ConfigMap '%s' in namespace '%s': %v", i.ConfigMap, i.Namespace, err)
		}
		fmt.Printf("ConfigMap '%s' updated\n", i.ConfigMap)
	}
}
