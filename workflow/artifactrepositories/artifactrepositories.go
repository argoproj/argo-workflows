package artifactrepositories

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/util/retry"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
)

type Interface interface {
	// Resolve Figures out the correct repository to for a workflow.
	Resolve(ctx context.Context, ref *wfv1.ArtifactRepositoryRef, workflowNamespace string) (*wfv1.ArtifactRepositoryRefStatus, error)
	// Get returns the referenced repository. May return nil (if no default artifact repository is configured).
	Get(ctx context.Context, ref *wfv1.ArtifactRepositoryRefStatus) (*wfv1.ArtifactRepository, error)
}

func New(kubernetesInterface kubernetes.Interface, namespace string, defaultArtifactRepository *wfv1.ArtifactRepository) Interface {
	return &artifactRepositories{kubernetesInterface, namespace, defaultArtifactRepository}
}

type artifactRepositories struct {
	kubernetesInterface       kubernetes.Interface
	namespace                 string
	defaultArtifactRepository *wfv1.ArtifactRepository
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
			{Default: true},
		}
	}
	for _, r := range refs {
		resolvedRef, err := s.get(ctx, r)
		if err != nil && (apierr.IsNotFound(err) || strings.Contains(err.Error(), "config map missing key")) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf(`error getting config map for artifact repository ref "%v": %w`, r, err)
		}
		logging.RequireLoggerFromContext(ctx).WithField("artifactRepositoryRef", r).Info(ctx, "resolved artifact repository")
		return resolvedRef, nil
	}
	return nil, fmt.Errorf(`failed to find any artifact repository for artifact repository ref "%v"`, ref)
}

func (s *artifactRepositories) Get(ctx context.Context, ref *wfv1.ArtifactRepositoryRefStatus) (*wfv1.ArtifactRepository, error) {
	ref, err := s.get(ctx, ref)
	if err != nil {
		return nil, err
	}
	return ref.ArtifactRepository, nil
}

func (s *artifactRepositories) get(ctx context.Context, ref *wfv1.ArtifactRepositoryRefStatus) (*wfv1.ArtifactRepositoryRefStatus, error) {
	if ref.ArtifactRepository != nil {
		return ref, nil
	}
	if ref.Default {
		return &wfv1.ArtifactRepositoryRefStatus{
			ArtifactRepositoryRef: ref.ArtifactRepositoryRef,
			Namespace:             ref.Namespace,
			Default:               true,
			ArtifactRepository:    s.defaultArtifactRepository,
		}, nil
	}
	var cm *v1.ConfigMap
	namespace := ref.Namespace
	configMap := ref.GetConfigMapOr("artifact-repositories")
	err := waitutil.Backoff(retry.DefaultRetry(ctx), func() (bool, error) {
		var err error
		cm, err = s.kubernetesInterface.CoreV1().ConfigMaps(namespace).Get(ctx, configMap, metav1.GetOptions{})
		return !errorsutil.IsTransientErrQuiet(ctx, err), err
	})
	if err != nil {
		return nil, err
	}
	key := ref.GetKeyOr(cm.Annotations["workflows.argoproj.io/default-artifact-repository"])
	value, ok := cm.Data[key]
	if !ok {
		return nil, fmt.Errorf(`config map missing key "%s" for artifact repository ref "%v"`, key, ref)
	}
	repo := &wfv1.ArtifactRepository{}
	if err := yaml.Unmarshal([]byte(value), repo); err != nil {
		return nil, fmt.Errorf(`failed to unmarshall config map key %q for artifact repository ref "%v": %w`, key, ref, err)
	}
	// we need the fully filled out ref so we can store it in the workflow status and it will never change
	// (even if the config map default annotation is changed)
	// this means users can change the default
	return &wfv1.ArtifactRepositoryRefStatus{
		Namespace:             namespace,
		ArtifactRepositoryRef: wfv1.ArtifactRepositoryRef{ConfigMap: configMap, Key: key},
		ArtifactRepository:    repo,
	}, nil
}
