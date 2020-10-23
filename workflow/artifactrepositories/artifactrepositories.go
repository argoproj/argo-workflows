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
	"github.com/argoproj/argo/util/retry"
)

type Interface interface {
	// Figures out the correct repository to for a workflow. This maybe a zero-valued repository - indicating you should be using
	// the default.
	ResolveArtifactRepositoryByRef(r *wfv1.ArtifactRepositoryRef, workflowNamespace string) (*wfv1.ArtifactRepositoryRef, error)
	// GetArtifactRepositoryByRef returns the referenced repository. May return nil.
	GetArtifactRepositoryByRef(r *wfv1.ArtifactRepositoryRef) (*config.ArtifactRepository, error)
}

func New() Interface {
	return &artifactRepositories{}
}

type artifactRepositories struct {
	kubernetesInterface       kubernetes.Interface
	managedNamespace          string
	defaultArtifactRepository *config.ArtifactRepository
}

func (s *artifactRepositories) ResolveArtifactRepositoryByRef(artifactRepositoryRef *wfv1.ArtifactRepositoryRef, workflowNamespace string) (*wfv1.ArtifactRepositoryRef, error) {
	for _, r := range []*wfv1.ArtifactRepositoryRef{
		artifactRepositoryRef,
		{Namespace: workflowNamespace},
		{Namespace: s.managedNamespace},
	} {
		err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
			var err error
			_, err = s.kubernetesInterface.CoreV1().ConfigMaps(r.Namespace).Get(r.GetConfigMap(), metav1.GetOptions{})
			return err == nil || apierr.IsNotFound(err), err
		})
		if apierr.IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf(`error getting config map for artifact repository ref "%v": %w`, artifactRepositoryRef, err)
		}
		log.WithField("artifactRepositoryRef", artifactRepositoryRef).Debug("found artifact repository by ref")
		return artifactRepositoryRef, nil
	}
	return wfv1.DefaultArtifactRepositoryRef, nil
}

func (s *artifactRepositories) GetArtifactRepositoryByRef(r *wfv1.ArtifactRepositoryRef) (*config.ArtifactRepository, error) {
	if r == wfv1.DefaultArtifactRepositoryRef {
		return s.defaultArtifactRepository, nil
	}
	var cm *v1.ConfigMap
	err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		var err error
		cm, err = s.kubernetesInterface.CoreV1().ConfigMaps(r.Namespace).Get(r.GetConfigMap(), metav1.GetOptions{})
		return err == nil || apierr.IsNotFound(err), err
	})
	if err != nil {
		return nil, fmt.Errorf(`failed to get config map for artifact repositry ref "%v": %w`, r, err)
	}
	value, ok := cm.Data[r.GetKey()]
	if !ok {
		return nil, fmt.Errorf(`config map missing key for artifact repositry ref "%v"`, r)
	}
	ar := &config.ArtifactRepository{}
	return ar, yaml.Unmarshal([]byte(value), ar)
}
