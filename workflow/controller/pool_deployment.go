package controller

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// This file builds a hot worker pool's Deployment from the spec.pool of its
// pooled TaskSets. The build reads no live state and makes no API calls, so a
// spec.pool always gives the same Deployment. Its pods follow the pod builder
// step by step, through the same derivations (workflowpod_derive.go), so
// their pod-level fields are what a pod for the template gets. Where they
// differ:
//
//   - The containers are worker (argoexec worker, in the aux container's
//     place), main (argoexec emissary --loop, with no command of its own) and
//     one sidecar per artifact plugin. Each task's command, args, plain env and
//     identity reach main and worker through its TaskSet, never the Deployment.
//   - Inputs are staged as in the init-less layout whatever the pool's layout:
//     the whole input-artifacts volume on main, which links each task's inputs
//     into place, and a plugin sidecar for every plugin, input or output,
//     because a pool's init containers run once per pod, not per task. Which
//     inputs, outputs and plugins a task uses varies within a pool, so the
//     volumes and mounts for them are always present.
//   - The legacy layout's init container only installs argoexec.
//   - Artifact secrets, the executor token and the kubeconfig are mounted only
//     in worker: no other container calls the API or reads artifact
//     secrets.
//   - Pods carry the pool's labels instead of the workflow's, and none of a
//     pod's per-task annotations, finalizer or activeDeadlineSeconds: the
//     worker enforces each task's deadlines, as the kubelet does a pod's.

// poolReservedLabels can't be set on a pool's pods through podMetadata or the
// template's metadata. The pool controller and the pool pod informer select
// pool pods by the pool and instance ID labels, and a pod with the workflow
// label would be taken for a workflow pod and missed by the pool pod informer.
var poolReservedLabels = []string{
	common.LabelKeyPoolName,
	common.LabelKeyPoolHash,
	common.LabelKeyControllerInstanceID,
	common.LabelKeyWorkflow,
}

// poolDeploymentName is the name of a pool's Deployment: the pool name and the
// first 10 characters of its hash.
func poolDeploymentName(name, hash string) string {
	return name + "-" + hash[:10]
}

// buildPoolDeployment builds the Deployment of the pool with hash, the
// pool-hash label of its TaskSets, in namespace from its spec.pool alone:
// dispatch looks up the image entrypoints of its artifact plugins into
// spec.pool. A spec it can't build fails the build, rather than building
// something else that the TaskSet's spec.pool didn't ask for. Replicas are
// spec.InitialReplicas, which apply only when the pool is created.
func buildPoolDeployment(namespace, hash string, spec *wfv1.PoolSpec) (*appsv1.Deployment, error) {
	if err := checkPoolSpec(hash, spec); err != nil {
		return nil, err
	}
	// The derivations below share the slices and maps they are given.
	spec = spec.DeepCopy()
	tmpl := spec.Template
	wfSpec := spec.WorkflowSpec
	if wfSpec == nil {
		wfSpec = &wfv1.WorkflowSpec{}
	}
	layout := newPodLayout(spec.InitlessPod)
	kubeConfig := poolSpecKubeConfig(spec.KubeConfig)
	logOpts := spec.ExecutorLogArgs
	// execCfg is for the argoexec containers that don't call the API: no
	// kubeconfig and no executor token. Each container gets its own copy of
	// the executor config, as buildExecContainer appends to its args.
	execCfg := func() execContainerConfig {
		executor := &apiv1.Container{}
		if spec.Executor != nil {
			executor = spec.Executor.DeepCopy()
		}
		return execContainerConfig{
			image:           spec.ExecutorImage,
			imagePullPolicy: spec.ExecutorImagePullPolicy,
			executor:        executor,
			instanceID:      spec.InstanceID,
		}
	}

	podSpec := apiv1.PodSpec{
		RestartPolicy:                 apiv1.RestartPolicyAlways,
		TerminationGracePeriodSeconds: &spec.TerminationGracePeriodSeconds,
		Volumes:                       podVolumes(kubeConfig, tmpl),
		ImagePullSecrets:              wfSpec.ImagePullSecrets,
	}
	podSpec.Volumes = append(podSpec.Volumes, layout.extraVolumes(spec.ExecutorImage, spec.ExecutorImagePullPolicy)...)
	if wfSpec.HostNetwork != nil {
		podSpec.HostNetwork = *wfSpec.HostNetwork
	}
	applyDNSConfig(&podSpec, wfSpec)
	podServiceAccount{
		serviceAccountName:           spec.ServiceAccountName,
		automountServiceAccountToken: spec.AutomountServiceAccountToken,
	}.applyTo(&podSpec)
	if spec.ExecutorServiceAccountTokenSecretName != "" {
		podSpec.Volumes = append(podSpec.Volumes, executorTokenVolume(spec.ExecutorServiceAccountTokenSecretName))
	}

	workerCfg := execCfg()
	workerCfg.kubeConfig = kubeConfig
	worker := poolWorkerContainer(tmpl, workerCfg, spec, logOpts)
	main, err := poolMainContainer(tmpl, spec, layout.argoexecBinaryPath(), logOpts)
	if err != nil {
		return nil, err
	}
	podSpec.InitContainers = layout.poolInitContainers(tmpl, execCfg(), logOpts)

	applySchedulingConstraints(&podSpec, wfSpec, tmpl, nil)
	applyPodResources(&podSpec, wfSpec, tmpl)
	podMeta := metav1.ObjectMeta{
		Labels:      map[string]string{},
		Annotations: map[string]string{common.AnnotationKeyDefaultContainer: common.MainContainerName},
	}
	applyPodMetadata(&podMeta, wfSpec, tmpl)
	for _, k := range poolReservedLabels {
		delete(podMeta.Labels, k)
	}
	maps.Copy(podMeta.Labels, dispatch.PoolLabels(spec, hash))

	// The volumes and mounts addVolumeReferences adds, the plugins' sockets,
	// then the input artifacts' and script's volumes, then the plugin
	// sidecars, each as for an init-less pod.
	podSpec.Volumes = append(podSpec.Volumes, wfSpec.Volumes...)
	secretVolumes, secretMounts := poolArtifactSecretVolumes(spec.ArtifactSecrets)
	podSpec.Volumes = append(podSpec.Volumes, secretVolumes...)
	worker.VolumeMounts = append(worker.VolumeMounts, secretMounts...)
	for _, p := range spec.ArtifactPlugins {
		podSpec.Volumes = append(podSpec.Volumes, p.Name.Volume())
	}

	podSpec.Volumes = append(podSpec.Volumes, volumeInputArtifacts)
	worker.VolumeMounts = append(worker.VolumeMounts, inputArtifactExecutorMounts(tmpl)...)
	main.VolumeMounts = append(main.VolumeMounts, volumeMountInputArtifacts)

	// A pod has the staging volume only for a script, but worker creates a
	// staging directory for every task, so a pool always has it.
	podSpec.Volumes = append(podSpec.Volumes, volumeScriptStaging)
	worker.VolumeMounts = append(worker.VolumeMounts, volumeMountScriptStaging)
	main.VolumeMounts = append(main.VolumeMounts, volumeMountScriptStaging)

	var sidecars []apiv1.Container
	for _, p := range spec.ArtifactPlugins {
		sidecar := buildArtifactSidecarContainer(tmpl, execCfg(), p.Name, p.Image, p.Command, layout.argoexecBinaryPath(), logOpts)
		withoutWorkflowUID(sidecar)
		layout.decorateUserContainer(sidecar, false)
		sidecar.VolumeMounts = append(sidecar.VolumeMounts, inputArtifactExecutorMounts(tmpl)...)
		worker.VolumeMounts = append(worker.VolumeMounts, p.Name.VolumeMount())
		sidecars = append(sidecars, *sidecar)
	}

	// As addOutputArtifactsVolumes: worker and the plugin sidecars see main's
	// mounts, under /mainctrfs, to collect outputs written to volumes.
	mirrorMainMounts(worker, main.VolumeMounts, tmpl)
	for i := range sidecars {
		mirrorMainMounts(&sidecars[i], main.VolumeMounts, tmpl)
	}
	podSpec.Containers = slices.Concat([]apiv1.Container{*worker, main}, sidecars)

	// The per-container finishing touches of the pod builder's last loops,
	// including each container's name in its env, which buildArtifactContainer
	// has given the plugin sidecars already.
	for i := range podSpec.Containers {
		c := &podSpec.Containers[i]
		if c.Name == common.MainContainerName {
			layout.decorateUserContainer(c, false)
		}
		if !common.IsArtifactPluginSidecar(c.Name) {
			c.Env = append(c.Env, containerNameEnv(c.Name))
		}
		// Worker and every plugin sidecar share the output directory: any
		// plugin may store a task's outputs.
		finishContainer(c, i, spec.ExecutorImage, len(spec.ArtifactPlugins) > 0 && c.Name != common.MainContainerName, podSpec.TerminationGracePeriodSeconds)
	}
	for i := range podSpec.InitContainers {
		c := &podSpec.InitContainers[i]
		c.VolumeMounts = append(c.VolumeMounts, volumeMountVarArgo)
		c.Env = append(c.Env, containerNameEnv(c.Name))
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      poolDeploymentName(spec.Name, hash),
			Namespace: namespace,
			Labels:    dispatch.PoolLabels(spec, hash),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &spec.InitialReplicas,
			Selector: &metav1.LabelSelector{MatchLabels: dispatch.PoolLabels(spec, hash)},
			Template: apiv1.PodTemplateSpec{ObjectMeta: podMeta, Spec: podSpec},
		},
	}, nil
}

// checkPoolSpec fails on a spec.pool buildPoolDeployment can't build.
func checkPoolSpec(hash string, spec *wfv1.PoolSpec) error {
	switch {
	case spec == nil:
		return fmt.Errorf("no pool spec")
	case len(hash) < 10 || len(validation.IsValidLabelValue(hash)) > 0:
		return fmt.Errorf("invalid pool hash %q", hash)
	case spec.Template == nil || (spec.Template.Container == nil && spec.Template.Script == nil):
		return fmt.Errorf("pool %q has no container or script template", spec.Name)
	case spec.ExecutorImage == "":
		return fmt.Errorf("pool %q has no executor image", spec.Name)
	}
	if errs := validation.IsDNS1123Label(poolDeploymentName(spec.Name, hash)); len(errs) > 0 {
		return fmt.Errorf("invalid pool name %q: %v", spec.Name, errs)
	}
	return nil
}

// poolWorkerContainer is a pool pod's worker: `argoexec worker`, which
// claims the pool's tasks, stages their inputs and reports their results. It
// takes the aux container's place and, beyond the mounts every argoexec
// container gets from cfg, mounts the executor token when the pool has one.
func poolWorkerContainer(tmpl *wfv1.Template, cfg execContainerConfig, spec *wfv1.PoolSpec, logOpts []string) *apiv1.Container {
	worker := buildExecContainer(common.WorkerContainerName, tmpl, cfg)
	worker.Command = append([]string{"argoexec", "worker"}, logOpts...)
	withoutWorkflowUID(worker)
	if spec.ExecutorServiceAccountTokenSecretName != "" {
		worker.VolumeMounts = append(worker.VolumeMounts, executorTokenVolumeMount())
	}
	worker.Env = append(worker.Env,
		apiv1.EnvVar{Name: common.EnvVarPoolHash, ValueFrom: &apiv1.EnvVarSource{
			FieldRef: &apiv1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "metadata.labels['" + common.LabelKeyPoolHash + "']"},
		}},
		apiv1.EnvVar{Name: common.EnvVarReleaseReserveSeconds, Value: fmt.Sprint(spec.ReleaseReserveSeconds)},
	)
	return worker
}

// poolMainContainer is a pool pod's main: the template's container with the
// controller's main container defaults merged in, as a pod's, running
// `argoexec emissary --loop` from argoexecPath, which runs each task's
// command as the worker hands it over. Its env is what spec.pool keeps of the
// template's (the valueFrom entries and envFrom, which the kubelet resolves
// at pod start) and the defaults'.
func poolMainContainer(tmpl *wfv1.Template, spec *wfv1.PoolSpec, argoexecPath string, logOpts []string) (apiv1.Container, error) {
	var ctrs []apiv1.Container
	switch {
	case tmpl.Container != nil:
		ctrs = []apiv1.Container{*tmpl.Container}
	case tmpl.Script != nil:
		ctrs = []apiv1.Container{tmpl.Script.Container}
	}
	if err := applyMainContainerDefaults(ctrs, tmpl, spec.MainContainer); err != nil {
		return apiv1.Container{}, err
	}
	main := ctrs[0]
	main.Command = append([]string{argoexecPath, "emissary", "--loop"}, logOpts...)
	main.Args = nil
	main.Env = append(main.Env, apiv1.EnvVar{Name: common.EnvVarReleaseReserveSeconds, Value: fmt.Sprint(spec.ReleaseReserveSeconds)})
	return main, nil
}

// withoutWorkflowUID drops the ARGO_WORKFLOW_UID executorEnvVars gives every
// argoexec container: a pool's pods serve many workflows, so no per-workflow
// env belongs on them.
func withoutWorkflowUID(c *apiv1.Container) {
	c.Env = slices.DeleteFunc(c.Env, func(e apiv1.EnvVar) bool { return e.Name == common.EnvVarWorkflowUID })
}

// poolArtifactSecretVolumes are the volumes of a pool's artifact secrets and
// worker's mounts of them, as createSecretVolumesAndMounts makes a pod's: a
// volume per Secret, holding the keys used. They are in Secret name order, not
// map order, so a spec.pool always gives the same pods.
func poolArtifactSecretVolumes(secrets []apiv1.SecretKeySelector) ([]apiv1.Volume, []apiv1.VolumeMount) {
	volumes := make(map[string]apiv1.Volume)
	keys := make(map[string]bool)
	for i := range secrets {
		createSecretVal(volumes, &secrets[i], keys)
	}
	vols, mounts := secretVolumesAndMountsFromMap(volumes)
	slices.SortFunc(vols, func(a, b apiv1.Volume) int { return cmp.Compare(a.Name, b.Name) })
	slices.SortFunc(mounts, func(a, b apiv1.VolumeMount) int { return cmp.Compare(a.Name, b.Name) })
	return vols, mounts
}

// poolSpecKubeConfig is the controller kubeconfig spec.pool carries, or nil.
func poolSpecKubeConfig(kc *wfv1.PoolKubeConfig) *config.KubeConfig {
	if kc == nil {
		return nil
	}
	return &config.KubeConfig{SecretName: kc.SecretName, SecretKey: kc.SecretKey, VolumeName: kc.VolumeName, MountPath: kc.MountPath}
}
