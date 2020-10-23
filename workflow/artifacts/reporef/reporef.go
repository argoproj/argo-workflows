package reporef

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/config"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func GetArtifactRepositoryByRef(kubernetesInterface kubernetes.Interface, arRef *wfv1.ArtifactRepositoryRef, namespaces ...string) (*config.ArtifactRepository, error) {
	for _, namespace := range namespaces {
		cm, err := kubernetesInterface.CoreV1().ConfigMaps(namespace).Get(arRef.GetConfigMap(), metav1.GetOptions{})
		if err != nil {
			if apierr.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		value, ok := cm.Data[arRef.Key]
		if !ok {
			continue
		}
		log.WithFields(log.Fields{"namespace": namespace, "name": cm.Name}).Debug("Found artifact repository by ref")
		ar := &config.ArtifactRepository{}
		err = yaml.Unmarshal([]byte(value), ar)
		if err != nil {
			return nil, err
		}
		return ar, nil
	}
	return nil, fmt.Errorf("failed to find artifactory ref {%s}/%s#%s", strings.Join(namespaces, ","), arRef.GetConfigMap(), arRef.Key)
}
