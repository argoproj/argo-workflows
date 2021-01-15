package artifactrepositories

import (
	"context"
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

//go:generate mockery -name Interface

type Interface interface {
	// Resolve Figures out the correct repository to for a workflow.
	Resolve(ctx context.Context, ref *wfv1.ArtifactRepositoryRef, workflowNamespace string) (*wfv1.ArtifactRepositoryRefStatus, error)
	// Get returns the referenced repository. May return nil (if no default artifact repository is configured).
	Get(ctx context.Context, ref *wfv1.ArtifactRepositoryRefStatus) (*config.ArtifactRepository, error)
}

func New(kubernetesInterface kubernetes.Interface, namespace string, defaultArtifactRepository *config.ArtifactRepository) Interface {
	return &artifactRepositories{kubernetesInterface, namespace, defaultArtifactRepository}
}

type artifactRepositories struct {
	kubernetesInterface       kubernetes.Interface
	namespace                 string
	defaultArtifactRepository *config.ArtifactRepository
}

func (s *artifactRepositories) Resolve(ctx context.Context, ref *wfv1.ArtifactRepositoryRef, workflowNamespace string) (*wfv1.ArtifactRepositoryRefStatus, error) {
	var refs []*wfv1.ArtifactRepositoryRefStatus
	if ref != nil {
		refs = []*wfv1.ArtifactRepositoryRefStatus{
			{Namespace: workflowNamespace, ArtifactRepositoryRef: wfv1.ArtifactRepositoryRef{ConfigMap: ref.ConfigMap, Key: ref.Key}},
			{Namespace: s.namespace, ArtifactRepositoryRef: wfv1.ArtifactRepositoryRef{ConfigMap: ref.ConfigMap, Key: ref.Key}},
		}
	} else {
		refs = []*wfv1.ArtifactRepositoryRefStatus{
			{Namespace: workflowNamespace},
			wfv1.DefaultArtifactRepositoryRefStatus,
		}
	}
	for _, r := range refs {
		resolvedRef, _, err := s.get(ctx, r)
		if apierr.IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf(`error getting config map for artifact repository ref "%v": %w`, r, err)
		}
		log.WithField("artifactRepositoryRef", r).Info("resolved artifact repository")
		return resolvedRef, nil
	}
	return nil, fmt.Errorf("failed to find any artifact repository - should never happen")
}

func (s *artifactRepositories) Get(ctx context.Context, ref *wfv1.ArtifactRepositoryRefStatus) (*config.ArtifactRepository, error) {
	_, repo, err := s.get(ctx, ref)
	return repo, err
}

func (s *artifactRepositories) get(ctx context.Context, ref *wfv1.ArtifactRepositoryRefStatus) (*wfv1.ArtifactRepositoryRefStatus, *config.ArtifactRepository, error) {
	if ref.Default {
		return ref, s.defaultArtifactRepository, nil
	}
	var cm *v1.ConfigMap
	namespace := ref.Namespace
	configMap := ref.GetConfigMapOr("artifact-repositories")
	err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		var err error
		cm, err = s.kubernetesInterface.CoreV1().ConfigMaps(namespace).Get(ctx, configMap, metav1.GetOptions{})
		return err == nil || !errorsutil.IsTransientErr(err), err
	})
	if err != nil {
		return nil, nil, err
	}
	key := ref.GetKeyOr(cm.Annotations["workflows.argoproj.io/default-artifact-repository"])
	value, ok := cm.Data[key]
	if !ok {
		return nil, nil, fmt.Errorf(`config map missing key %s for artifact repository ref "%v"`, ref, key)
	}
	repo := &config.ArtifactRepository{}
	// we need the fully filled out ref so we can store it in the workflow status and it will never change
	// (even if the config map default annotation is changed)
	// this means users can change the default
	return &wfv1.ArtifactRepositoryRefStatus{Namespace: namespace, ArtifactRepositoryRef: wfv1.ArtifactRepositoryRef{ConfigMap: configMap, Key: key}}, repo, yaml.Unmarshal([]byte(value), repo)
}
