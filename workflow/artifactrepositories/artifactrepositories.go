package artifactrepositories

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/config"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo/util/errors"
	"github.com/argoproj/argo/util/retry"
)

type Interface interface {
	// Figures out the correct repository to for a workflow. This maybe a zero-valued repository - indicating you should be using
	// the default.
	ResolveArtifactRepositoryByRef(ref *wfv1.ArtifactRepositoryRef, workflowNamespace string) (*wfv1.ArtifactRepositoryRef, error)
	// GetArtifactRepositoryByRef returns the referenced repository. May return nil.
	GetArtifactRepositoryByRef(ref *wfv1.ArtifactRepositoryRef) (*config.ArtifactRepository, error)
}

func New(kubernetesInterface kubernetes.Interface, managedNamespace string, defaultArtifactRepository *config.ArtifactRepository) Interface {
	return &artifactRepositories{kubernetesInterface, managedNamespace, defaultArtifactRepository}
}

type artifactRepositories struct {
	kubernetesInterface       kubernetes.Interface
	managedNamespace          string
	defaultArtifactRepository *config.ArtifactRepository
}

func (s *artifactRepositories) ResolveArtifactRepositoryByRef(ref *wfv1.ArtifactRepositoryRef, workflowNamespace string) (*wfv1.ArtifactRepositoryRef, error) {
	// if this is explicit, it is our preference and must exist
	if ref != nil {
		return &wfv1.ArtifactRepositoryRef{
			Namespace: ref.GetNamespaceOr(workflowNamespace),
			ConfigMap: ref.GetConfigMap(),
			Key:       ref.GetKey(),
		}, nil
	}

	for _, r := range []*wfv1.ArtifactRepositoryRef{{Namespace: workflowNamespace}, {Namespace: s.managedNamespace}, wfv1.DefaultArtifactRepositoryRef} {
		_, err := s.GetArtifactRepositoryByRef(r)
		if apierr.IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf(`error getting config map for artifact repository ref "%v": %w`, r, err)
		}
		log.WithField("ref", r).Debug("found artifact repository by ref")
		return r, nil
	}
	return nil, fmt.Errorf("failed to find any artifact repository - should never happen")
}

func (s *artifactRepositories) GetArtifactRepositoryByRef(ref *wfv1.ArtifactRepositoryRef) (*config.ArtifactRepository, error) {
	if ref == wfv1.DefaultArtifactRepositoryRef {
		return s.defaultArtifactRepository, nil
	}
	var cm *v1.ConfigMap
	err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		var err error
		cm, err = s.kubernetesInterface.CoreV1().ConfigMaps(ref.Namespace).Get(ref.GetConfigMap(), metav1.GetOptions{})
		return err == nil || !errorsutil.IsTransientErr(err), err
	})
	if err != nil {
		return nil, err
	}
	value, ok := cm.Data[ref.GetKey()]
	if !ok {
		return nil, fmt.Errorf(`config map missing key for artifact repository ref "%v"`, ref)
	}
	ar := &config.ArtifactRepository{}
	return ar, yaml.Unmarshal([]byte(value), ar)
}
