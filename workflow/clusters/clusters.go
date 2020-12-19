package clusters

import (
	"encoding/json"
	"fmt"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo/config/clusters"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func GetConfigs(restConfig *rest.Config, kubeclientset kubernetes.Interface, thisClusterName wfv1.ClusterName, namespace string) (map[string]*rest.Config, map[wfv1.ClusterName]kubernetes.Interface, error) {
	restConfigs := map[string]*rest.Config{}
	if restConfig != nil {
		restConfigs[thisClusterName] = restConfig
	}
	kubernetesInterfaces := map[wfv1.ClusterName]kubernetes.Interface{thisClusterName: kubeclientset}
	secret, err := kubeclientset.CoreV1().Secrets(namespace).Get("clusters", metav1.GetOptions{})
	if apierr.IsNotFound(err) {
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get secret/clusters: %w", err)
	} else {
		for clusterName, data := range secret.Data {
			c := &clusters.Config{}
			err := json.Unmarshal(data, c)
			if err != nil {
				return nil, nil, fmt.Errorf("failed unmarshall JSON for cluster %s: %w", clusterName, err)
			}
			restConfigs[clusterName] = c.RestConfig()
			clientset, err := kubernetes.NewForConfig(restConfigs[clusterName])
			if err != nil {
				return nil, nil, fmt.Errorf("failed create new kube client for cluster %s: %w", clusterName, err)
			}
			kubernetesInterfaces[clusterName] = clientset
		}
	}
	return restConfigs, kubernetesInterfaces, nil
}
