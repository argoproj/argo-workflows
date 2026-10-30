package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-workflows/v4"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v4/util"
	"github.com/argoproj/argo-workflows/v4/util/env"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/logs"
	"github.com/argoproj/argo-workflows/v4/util/retry"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/emissary"
	executortracing "github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
	"github.com/argoproj/argo-workflows/v4/workflow/tracing"
)

// InitProcess builds the process-wide executor state from the pod's
// environment: API clients, tracing, namespace and pod-scoped settings.
// It runs once per process; the per-task half is TaskConfigFromEnv.
func InitProcess(ctx context.Context, clientConfig clientcmd.ClientConfig) *executor.Process {
	process, _, _ := initProcess(ctx, clientConfig, nil)
	return process
}

// initProcess is InitProcess, which also returns the API client's config
// and the namespace, for a process that needs more clients. adjust, if
// set, changes the process's config, as read from the environment, before
// the process is built.
//
//nolint:contextcheck
func initProcess(ctx context.Context, clientConfig clientcmd.ClientConfig, adjust func(*executor.ProcessConfig)) (*executor.Process, *restclient.Config, string) {
	version := argo.GetVersion()
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithFields(version.Fields()).Info(ctx, "Starting Workflow Executor")
	config, err := clientConfig.ClientConfig()
	CheckErr(err)
	config = restclient.AddUserAgent(config, fmt.Sprintf("argo-workflows/%s argo-executor", version.Version))

	//nolint:contextcheck
	bgCtx := logger.NewBackgroundContext()
	logs.AddK8SLogTransportWrapper(bgCtx, config) // lets log all request as we should typically do < 5 per pod, so this is will show up problems
	tracing.AddTracingTransportWrapper(bgCtx, config)

	namespace, _, err := clientConfig.Namespace()
	CheckErr(err)

	clientset, err := kubernetes.NewForConfig(config)
	CheckErr(err)

	restClient := clientset.RESTClient()

	podName, ok := os.LookupEnv(common.EnvVarPodName)
	if !ok {
		logger.Error(ctx, fmt.Sprintf("Unable to determine pod name from environment variable %s", common.EnvVarPodName))
		os.Exit(1)
	}

	cfg := executor.ProcessConfig{
		PodName:                    podName,
		PodUID:                     types.UID(os.Getenv(common.EnvVarPodUID)),
		Namespace:                  namespace,
		InstanceID:                 os.Getenv(common.EnvVarInstanceID),
		TerminationGracePeriod:     TerminationGracePeriodDuration(),
		ArtifactPluginNames:        os.Getenv(common.EnvVarArtifactPluginNames),
		RemoveLocalArtPath:         os.Getenv("REMOVE_LOCAL_ART_PATH") == "true",
		RetryBackoff:               retry.ExecutorRetry(ctx),
		ResourceStateCheckInterval: env.LookupEnvDurationOr(ctx, "RESOURCE_STATE_CHECK_INTERVAL", 5*time.Second),
		TaskResultRetryTimeout:     retry.ExecutorTaskResultRetryTimeout(ctx),
	}
	if adjust != nil {
		adjust(&cfg)
	}
	process, err := executor.NewProcess(
		ctx,
		clientset,
		versioned.NewForConfigOrDie(config).ArgoprojV1alpha1().WorkflowTaskResults(namespace),
		restClient,
		func(paths executor.TaskPaths) executor.ContainerRuntimeExecutor {
			return emissary.New(paths.VarRunArgo)
		},
		cfg,
	)
	CheckErr(err)
	return process, config, namespace
}

// TaskConfigFromEnv reads the per-task values (template, node identity,
// deadline) that the controller baked into the pod for the single task a
// classic argoexec pod runs.
func TaskConfigFromEnv(varRunArgo string) executor.TaskConfig {
	cfg, err := TaskConfigFromLookup(os.LookupEnv, varRunArgo)
	CheckErr(err)
	return cfg
}

// TaskConfigFromLookup is TaskConfigFromEnv with the environment read
// through lookupEnv.
func TaskConfigFromLookup(lookupEnv func(string) (string, bool), varRunArgo string) (executor.TaskConfig, error) {
	getenv := func(key string) string {
		v, _ := lookupEnv(key)
		return v
	}
	tmpl := &wfv1.Template{}
	envVarTemplateValue, ok := lookupEnv(common.EnvVarTemplate)
	var templateBytes []byte
	var err error
	if !ok {
		// Wait container reads template from the file written by init container,
		// not from the env var.
		templateBytes, err = os.ReadFile(varRunArgo + "/template")
	} else {
		// Offload-sentinel resolution is shared with the emissary via
		// common.ResolveTemplateEnvValue so the offload protocol stays in one place.
		templateBytes, err = common.ResolveTemplateEnvValue(envVarTemplateValue, common.EnvConfigMountPath)
	}
	if err != nil {
		return executor.TaskConfig{}, err
	}
	if err = json.Unmarshal(templateBytes, tmpl); err != nil {
		return executor.TaskConfig{}, err
	}

	deadline, err := time.Parse(time.RFC3339, getenv(common.EnvVarDeadline))
	if err != nil {
		return executor.TaskConfig{}, err
	}

	var inputPlugins []wfv1.ArtifactPluginName
	for _, p := range common.SplitPluginNames(getenv(common.EnvVarInputArtifactPluginNames)) {
		inputPlugins = append(inputPlugins, wfv1.ArtifactPluginName(p))
	}

	// errors ignored because values are set by the controller and checked there.
	progressPatchTickDuration, _ := time.ParseDuration(getenv(common.EnvVarProgressPatchTickDuration))
	progressFileTickDuration, _ := time.ParseDuration(getenv(common.EnvVarProgressFileTickDuration))

	return executor.TaskConfig{
		WorkflowName:              getenv(common.EnvVarWorkflowName),
		WorkflowUID:               types.UID(getenv(common.EnvVarWorkflowUID)),
		NodeID:                    getenv(common.EnvVarNodeID),
		Template:                  *tmpl,
		IncludeScriptOutput:       getenv(common.EnvVarIncludeScriptOutput) == "true",
		Deadline:                  deadline,
		ProgressFile:              getenv(common.EnvVarProgressFile),
		ProgressPatchTickDuration: progressPatchTickDuration,
		ProgressFileTickDuration:  progressFileTickDuration,
		TraceContext:              executortracing.TraceContextFromEnv(lookupEnv),
		InputArtifactPluginNames:  inputPlugins,
	}, nil
}

// Init builds the executor for the one task a classic argoexec pod runs:
// process-wide setup followed by the single task described by the
// environment.
func Init(ctx context.Context, clientConfig clientcmd.ClientConfig, varRunArgo string) *executor.WorkflowExecutor {
	process := InitProcess(ctx, clientConfig)
	task, _, _ := (&PodSource{VarRunArgo: varRunArgo}).Next(ctx) // yields once and never errors
	// This process only runs the resource or data action. The pod's plan,
	// and every read of main's files, is in its wait or supervisor
	// container, which takes its own layout, so the layout here is unused.
	return process.NewExecutor(task, executor.LayoutLegacy, podSink(process)(task))
}

// TerminationGracePeriodDuration returns the terminationGracePeriodSeconds of
// the pod spec in time.Duration format, parsed from the environment
// (ARGO_TERMINATION_GRACE_PERIOD_SECONDS).
func TerminationGracePeriodDuration() time.Duration {
	x, _ := strconv.ParseInt(os.Getenv(common.EnvVarTerminationGracePeriodSeconds), 10, 64)
	if x > 0 {
		return time.Duration(x) * time.Second
	}
	return 30 * time.Second
}

// ReleaseReserve is the time a hot worker pool pod's containers keep back at
// the end of the pod's grace period to release its task
// (ARGO_RELEASE_RESERVE_SECONDS), which the controller sets on both from
// its validated config.
func ReleaseReserve() (time.Duration, error) {
	const maxSeconds = math.MaxInt64 / int64(time.Second)
	v := os.Getenv(common.EnvVarReleaseReserveSeconds)
	reserve, err := strconv.ParseInt(v, 10, 64)
	if err != nil || reserve <= 0 || reserve > maxSeconds {
		return 0, fmt.Errorf("%s must be a whole number of seconds between 1 and %d, not %q", common.EnvVarReleaseReserveSeconds, maxSeconds, v)
	}
	return time.Duration(reserve) * time.Second, nil
}

// CheckErr is a convenience function to panic upon error
func CheckErr(err error) {
	if err != nil {
		util.WriteTerminateMessage(err.Error())
		panic(err.Error())
	}
}
