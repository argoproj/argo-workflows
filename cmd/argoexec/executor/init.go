package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-workflows/v4"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v4/util"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/logs"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/emissary"
	"github.com/argoproj/argo-workflows/v4/workflow/tracing"
)

//nolint:contextcheck
func Init(ctx context.Context, clientConfig clientcmd.ClientConfig, varRunArgo string) *executor.WorkflowExecutor {
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
		logger.WithFatal().Error(ctx, fmt.Sprintf("Unable to determine pod name from environment variable %s", common.EnvVarPodName))
		os.Exit(1)
	}

	tmpl := &wfv1.Template{}
	envVarTemplateValue, ok := os.LookupEnv(common.EnvVarTemplate)
	// wait container reads template from the file written by init container, instead of from environment variable.
	if !ok {
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/template")
		CheckErr(err)
		envVarTemplateValue = string(data)
	} else if envVarTemplateValue == common.EnvVarTemplateOffloaded {
		var data []byte
		data, err = os.ReadFile(filepath.Join(common.EnvConfigMountPath, common.EnvVarTemplate))
		CheckErr(err)
		envVarTemplateValue = string(data)
	}
	CheckErr(json.Unmarshal([]byte(envVarTemplateValue), tmpl))

	includeScriptOutput := os.Getenv(common.EnvVarIncludeScriptOutput) == "true"
	deadline, err := time.Parse(time.RFC3339, os.Getenv(common.EnvVarDeadline))
	CheckErr(err)

	// errors ignored because values are set by the controller and checked there.
	annotationPatchTickDuration, _ := time.ParseDuration(os.Getenv(common.EnvVarProgressPatchTickDuration))
	progressFileTickDuration, _ := time.ParseDuration(os.Getenv(common.EnvVarProgressFileTickDuration))

	cre, err := emissary.New()
	CheckErr(err)

	wfExecutor, err := executor.NewExecutor(
		ctx,
		clientset,
		versioned.NewForConfigOrDie(config).ArgoprojV1alpha1().WorkflowTaskResults(namespace),
		restClient,
		podName,
		types.UID(os.Getenv(common.EnvVarPodUID)),
		os.Getenv(common.EnvVarWorkflowName),
		types.UID(os.Getenv(common.EnvVarWorkflowUID)),
		os.Getenv(common.EnvVarNodeID),
		namespace,
		cre,
		*tmpl,
		includeScriptOutput,
		deadline,
		annotationPatchTickDuration,
		progressFileTickDuration,
	)
	CheckErr(err)

	logger.
		WithFields(version.Fields()).
		WithField("namespace", namespace).
		WithField("podName", podName).
		WithField("templateName", wfExecutor.Template.Name).
		WithField("includeScriptOutput", includeScriptOutput).
		WithField("deadline", deadline).
		Info(ctx, "Executor initialized")
	return &wfExecutor
}

// CheckErr is a convenience function to panic upon error
func CheckErr(err error) {
	if err != nil {
		util.WriteTerminateMessage(err.Error())
		panic(err.Error())
	}
}
