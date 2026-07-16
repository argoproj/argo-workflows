package utils

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/artifactrepositories"
)

// ResolveArtifactLocation resolves the default artifact repository for namespace and
// returns its location. It returns a nil location (with no error) if no default
// artifact repository is configured, matching artifactrepositories.Interface.Get.
func ResolveArtifactLocation(ctx context.Context, repositories artifactrepositories.Interface, ref *wfv1.ArtifactRepositoryRef, namespace string) (*wfv1.ArtifactLocation, error) {
	repoRef, err := repositories.Resolve(ctx, ref, namespace)
	if err != nil {
		return nil, err
	}
	repo, err := repositories.Get(ctx, repoRef)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, nil
	}
	return repo.ToArtifactLocation(), nil
}
