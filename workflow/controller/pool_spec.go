package controller

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	apiv1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/pool"
)

// poolSpecExtras are the inputs of a pooled task's spec.pool beyond the pod
// builder's snapshot: what the pod builder gets through podBuilderDeps, and
// what only pools need.
type poolSpecExtras struct {
	// executorLogArgs are the log flags of every argoexec container
	// (podBuilderDeps.getExecutorLogOpts).
	executorLogArgs []string
	// executorTokenSecretName is the Secret holding the executor service
	// account's token (podBuilderDeps.getServiceAccountTokenName), looked up by
	// the caller. It must be set when the task has an executor service account.
	executorTokenSecretName string
	// workflowTemplateName is the WorkflowTemplate or ClusterWorkflowTemplate
	// the template comes from, through templateRef or the workflow's
	// workflowTemplateRef; empty when there is none. It is part of the default
	// pool name.
	workflowTemplateName string
	// controllerVersion is the dispatching controller's version.
	controllerVersion string
}

// buildPoolSpec builds the spec.pool of a pooled task from the pod builder's
// snapshot of it (see podBuilderInputs) and extras: everything the pool's
// Deployment is built from, and nothing per task. It reads no live state and
// makes no API calls. Template and workflow spec fields are copied or left
// out as the pool package classifies them (pool.HashedTemplate and
// pool.HashedWorkflowSpec), and a rejected field set on either fails the
// build. The pool hash is poolHash of
// the result.
func buildPoolSpec(ctx context.Context, in *podBuilderInputs, x poolSpecExtras) (*wfv1.PoolSpec, error) {
	tmpl := in.tmpl.DeepCopy()
	if tmpl.Pool == nil {
		return nil, fmt.Errorf("template %q is not pooled", tmpl.Name)
	}
	foldBoundarySchedulingConstraints(tmpl, in.boundaryTemplate)
	wfSpec := in.execWfSpec.DeepCopy()

	spec := &wfv1.PoolSpec{
		Name:                          pool.Name(tmpl.Pool, x.workflowTemplateName, tmpl.Name),
		InitialReplicas:               1,
		RetentionSeconds:              in.config.HotWorkers.GetDefaultRetentionSeconds(),
		ArtifactRepository:            in.artifactRepository.DeepCopy(),
		InstanceID:                    in.execCfg.instanceID,
		ControllerVersion:             x.controllerVersion,
		Executor:                      nonZeroContainer(in.config.GetExecutor()),
		MainContainer:                 nonZeroContainer(in.mainContainerDefaults),
		ExecutorImage:                 in.execCfg.image,
		ExecutorImagePullPolicy:       in.execCfg.imagePullPolicy,
		ExecutorLogArgs:               slices.Clone(x.executorLogArgs),
		KubeConfig:                    poolKubeConfig(in),
		InitlessPod:                   in.config.InitlessPod.IsEnabled(),
		TerminationGracePeriodSeconds: in.config.HotWorkers.GetTerminationGracePeriodSeconds(),
		ReleaseReserveSeconds:         in.config.HotWorkers.GetReleaseReserveSeconds(),
	}
	// Both are checked by validation once the template is substituted.
	if r, ok, err := pool.InitialReplicas.Value(tmpl.Pool.InitialReplicas); err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "pool.%s", err)
	} else if ok {
		spec.InitialReplicas = int32(r)
	}
	if r, ok, err := pool.RetentionSeconds.Value(tmpl.Pool.RetentionSeconds); err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "pool.%s", err)
	} else if ok {
		spec.RetentionSeconds = r
	}

	var err error
	if spec.Template, err = pool.HashedTemplate(tmpl); err != nil {
		return nil, err
	}
	// After the copy, which names any other template type's field as rejected.
	if t := tmpl.GetType(); t != wfv1.TemplateTypeContainer && t != wfv1.TemplateTypeScript {
		return nil, errors.Errorf(errors.CodeBadRequest, "only container and script templates can be pooled, not %s", t)
	}
	if spec.WorkflowSpec, err = pool.HashedWorkflowSpec(wfSpec); err != nil {
		return nil, err
	}
	volumes, err := mountedWorkflowVolumes(tmpl, in.volumes)
	if err != nil {
		return nil, err
	}
	spec.WorkflowSpec.Volumes = volumes
	if reflect.ValueOf(*spec.WorkflowSpec).IsZero() {
		spec.WorkflowSpec = nil
	}

	sa, err := derivePodServiceAccount(tmpl, wfSpec)
	if err != nil {
		return nil, err
	}
	spec.ServiceAccountName = sa.serviceAccountName
	spec.AutomountServiceAccountToken = sa.automountServiceAccountToken
	if sa.executorServiceAccountName != "" {
		if x.executorTokenSecretName == "" {
			return nil, fmt.Errorf("no token Secret for executor service account %q", sa.executorServiceAccountName)
		}
		spec.ExecutorServiceAccountTokenSecretName = x.executorTokenSecretName
	}

	spec.ArtifactSecrets = poolArtifactSecrets(tmpl, in.artifactRepository)
	spec.ArtifactPlugins, err = poolArtifactPlugins(ctx, tmpl, in)
	if err != nil {
		return nil, err
	}
	return spec, nil
}

// poolHash is a pool's identity within its namespace: a hash of its
// spec.pool except initialReplicas and retentionSeconds. It is lowercase
// base32 without padding (52 characters), so it is a valid label value and
// any prefix of it is DNS-label safe. It is computed once, at dispatch, and
// stored in the pool-hash label; nothing recomputes it, so a change to
// PoolSpec or its serialisation only rolls pools, which a new controller
// version does anyway.
func poolHash(spec *wfv1.PoolSpec) (string, error) {
	hashed := *spec
	hashed.InitialReplicas = 0
	hashed.RetentionSeconds = 0
	// encoding/json writes struct fields in declaration order and map keys
	// sorted, so the serialisation is canonical for equal specs.
	data, err := json.Marshal(hashed)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:])), nil
}

// foldBoundarySchedulingConstraints gives tmpl the boundary template's node
// selector, affinity and tolerations where it has none of its own, so
// applySchedulingConstraints with no boundary template gives what it gives
// tmpl with this boundary template. boundaryTemplate may be nil.
func foldBoundarySchedulingConstraints(tmpl, boundaryTemplate *wfv1.Template) {
	if boundaryTemplate == nil {
		return
	}
	if len(tmpl.NodeSelector) == 0 && len(boundaryTemplate.NodeSelector) > 0 {
		tmpl.NodeSelector = boundaryTemplate.NodeSelector
	}
	if tmpl.Affinity == nil && boundaryTemplate.Affinity != nil {
		tmpl.Affinity = boundaryTemplate.Affinity
	}
	if len(tmpl.Tolerations) == 0 && len(boundaryTemplate.Tolerations) > 0 {
		tmpl.Tolerations = boundaryTemplate.Tolerations
	}
}

// mountedWorkflowVolumes returns the workflow volumes, as the controller
// substituted them, that the template mounts and doesn't define itself, in
// mount order: the workflow volumes a pod for the template gets
// (pool.MountedWorkflowVolumes).
func mountedWorkflowVolumes(tmpl *wfv1.Template, volumes []apiv1.Volume) ([]apiv1.Volume, error) {
	mounted, unresolved := pool.MountedWorkflowVolumes(tmpl, volumes)
	if len(unresolved) > 0 {
		return nil, errors.Errorf(errors.CodeBadRequest, "volume '%s' is neither the template's nor in the workflow's volumes; pooled templates can't mount volumeClaimTemplates volumes", unresolved[0])
	}
	return mounted, nil
}

// poolArtifactSecrets returns the secrets any task of the pool may read for its
// artifacts: the repository's, which any task may use for its outputs and logs,
// and those of the template's archive location and artifacts, the same ones a
// pod mounts (see createSecretVolumesAndMounts). They are sorted, without
// duplicates, and reduced to name and key, which is all a pod's secret volumes
// use, so the order of artifacts doesn't change the hash.
func poolArtifactSecrets(tmpl *wfv1.Template, repo *wfv1.ArtifactRepository) []apiv1.SecretKeySelector {
	var secrets []apiv1.SecretKeySelector
	for _, l := range append([]*wfv1.ArtifactLocation{repo.ToArtifactLocation()}, templateArtifactLocations(tmpl)...) {
		for _, s := range artifactLocationSecrets(l) {
			if s == nil || s.Name == "" || s.Key == "" {
				continue
			}
			secrets = append(secrets, apiv1.SecretKeySelector{LocalObjectReference: s.LocalObjectReference, Key: s.Key})
		}
	}
	compare := func(a, b apiv1.SecretKeySelector) int {
		return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.Key, b.Key))
	}
	slices.SortFunc(secrets, compare)
	return slices.CompactFunc(secrets, func(a, b apiv1.SecretKeySelector) bool { return compare(a, b) == 0 })
}

// poolArtifactPlugins returns the artifact driver plugins any task of the pool
// may use, sorted by name: the repository's, and those of the template's
// archive location and input and output artifacts, as a pod's plugin
// containers would be chosen (see buildPluginSidecars).
func poolArtifactPlugins(ctx context.Context, tmpl *wfv1.Template, in *podBuilderInputs) ([]wfv1.PoolArtifactPlugin, error) {
	names := slices.Concat(
		tmpl.Inputs.Artifacts.GetPluginNames(ctx, in.artifactRepository, wfv1.ExcludeLogs, tmpl.ArchiveLocation),
		tmpl.Outputs.Artifacts.GetPluginNames(ctx, in.artifactRepository, wfv1.IncludeLogs, tmpl.ArchiveLocation),
	)
	if in.artifactRepository != nil && in.artifactRepository.Plugin != nil {
		names = append(names, in.artifactRepository.Plugin.Name)
	}
	if tmpl.ArchiveLocation != nil && tmpl.ArchiveLocation.Plugin != nil {
		names = append(names, tmpl.ArchiveLocation.Plugin.Name)
	}
	slices.Sort(names)
	names = slices.Compact(names)
	var plugins []wfv1.PoolArtifactPlugin
	for _, name := range names {
		if name == "" {
			continue
		}
		driver, err := in.config.GetArtifactDriver(name)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, wfv1.PoolArtifactPlugin{Name: driver.Name, Image: driver.Image, ConnectionTimeoutSeconds: driver.ConnectionTimeoutSeconds})
	}
	return plugins, nil
}

// poolKubeConfig is the controller's kubeconfig for argoexec containers, with
// its default volume name and mount path filled in, or nil when none is
// configured.
func poolKubeConfig(in *podBuilderInputs) *wfv1.PoolKubeConfig {
	kc := in.execCfg.kubeConfig
	if kc == nil {
		return nil
	}
	return &wfv1.PoolKubeConfig{SecretName: kc.SecretName, SecretKey: kc.SecretKey, VolumeName: kubeConfigVolumeName(kc), MountPath: kubeConfigMountPath(kc)}
}

// nonZeroContainer copies c, or returns nil when c is nil or empty, which
// configure nothing alike.
func nonZeroContainer(c *apiv1.Container) *apiv1.Container {
	if c == nil || reflect.ValueOf(*c).IsZero() {
		return nil
	}
	return c.DeepCopy()
}
