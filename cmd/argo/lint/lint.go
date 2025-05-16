package lint

import (
	"context"
	"fmt"
	"io"
	"strings"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient"
	clusterworkflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	wf "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fileutil "github.com/argoproj/argo-workflows/v3/util/file"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type ServiceClients struct {
	WorkflowsClient               workflowpkg.WorkflowServiceClient
	WorkflowTemplatesClient       workflowtemplatepkg.WorkflowTemplateServiceClient
	CronWorkflowsClient           cronworkflowpkg.CronWorkflowServiceClient
	ClusterWorkflowTemplateClient clusterworkflowtemplatepkg.ClusterWorkflowTemplateServiceClient
}

type LintOptions struct {
	Files            []string
	Strict           bool
	DefaultNamespace string
	Formatter        Formatter
	ServiceClients   ServiceClients

	// Printer if not nil the lint result is written to this writer after each
	// file is linted.
	Printer io.Writer
}

// LintResult represents the result of linting objects from a single source
type LintResult struct {
	File   string
	Errs   []error
	Linted bool
}

// LintResults represents the result of linting objects from multiple sources
type LintResults struct {
	Results        []*LintResult
	Success        bool
	msg            string
	fmtr           Formatter
	anythingLinted bool
}

type Formatter interface {
	Format(*LintResult) string
	Summarize(*LintResults) string
}

var (
	defaultFormatter = formatterPretty{}

	formatters = map[string]Formatter{
		"pretty": formatterPretty{},
		"simple": formatterSimple{},
	}
)

func GetFormatter(fmtr string) (Formatter, error) {
	f, exists := formatters[fmtr]
	if !exists {
		return nil, fmt.Errorf("unknown formatter: %s", fmtr)
	}
	return f, nil
}

// RunLint lints the specified kinds in the specified files and prints the results to os.Stdout.
// If linting fails it will exit with status code 1.
func RunLint(ctx context.Context, client apiclient.Client, kinds []string, output string, offline bool, opts LintOptions) error {
	fmtr, err := GetFormatter(output)
	if err != nil {
		return err
	}
	clients, err := getLintClients(client, kinds)
	if err != nil {
		return err
	}
	opts.ServiceClients = clients
	opts.Formatter = fmtr
	res, err := Lint(ctx, &opts)
	if err != nil {
		return err
	}

	if !res.Success {
		log.StandardLogger().Exit(1)
	}
	return nil
}

// Lint reads all files, returns linting errors of all of the entities of the specified kinds.
// Entities of other kinds are ignored.
func Lint(ctx context.Context, opts *LintOptions) (*LintResults, error) {
	var fmtr Formatter = defaultFormatter
	var w = io.Discard
	if opts.Formatter != nil {
		fmtr = opts.Formatter
	}
	if opts.Printer != nil {
		w = opts.Printer
	}

	results := &LintResults{
		Results: []*LintResult{},
		fmtr:    fmtr,
	}

	for _, file := range opts.Files {
		err := fileutil.WalkManifests(file, func(path string, data []byte) error {
			res := lintData(ctx, path, data, opts)
			results.Results = append(results.Results, res)

			_, err := w.Write([]byte(results.fmtr.Format(res)))
			return err
		})
		if err != nil {
			return nil, err
		}
	}

	results.evaluate()
	_, err := w.Write([]byte(results.fmtr.Summarize(results)))
	return results, err
}

func lintData(ctx context.Context, src string, data []byte, opts *LintOptions) *LintResult {
	res := &LintResult{
		File: src,
		Errs: []error{},
	}

	for i, pr := range common.ParseObjects(data, opts.Strict) {
		obj, err := pr.Object, pr.Err
		if obj == nil {
			continue // could not parse to kubernetes object
		}
		// we should prefer the object's namespace
		namespace := obj.GetNamespace()
		if namespace == "" {
			namespace = opts.DefaultNamespace
		}
		objName := ""

		switch v := obj.(type) {
		case *wfv1.ClusterWorkflowTemplate:
			objName = getObjectName(wf.ClusterWorkflowTemplateKind, v, i)
			if opts.ServiceClients.ClusterWorkflowTemplateClient == nil {
				log.Debugf("ignoring %s, not in lint options", objName)
				continue
			}
			res.Linted = true
			if err == nil {
				_, err = opts.ServiceClients.ClusterWorkflowTemplateClient.LintClusterWorkflowTemplate(
					ctx,
					&clusterworkflowtemplatepkg.ClusterWorkflowTemplateLintRequest{Template: v},
				)
			}
		case *wfv1.CronWorkflow:
			objName = getObjectName(wf.CronWorkflowKind, v, i)
			if opts.ServiceClients.CronWorkflowsClient == nil {
				log.Debugf("ignoring %s, not in lint options kinds", objName)
				continue
			}
			res.Linted = true
			if err == nil {
				_, err = opts.ServiceClients.CronWorkflowsClient.LintCronWorkflow(
					ctx,
					&cronworkflowpkg.LintCronWorkflowRequest{Namespace: namespace, CronWorkflow: v},
				)
			}
		case *wfv1.Workflow:
			objName = getObjectName(wf.WorkflowKind, v, i)
			if opts.ServiceClients.WorkflowsClient == nil {
				log.Debugf("ignoring %s, not in lint options kinds", objName)
				continue
			}
			res.Linted = true
			if err == nil {
				_, err = opts.ServiceClients.WorkflowsClient.LintWorkflow(
					ctx,
					&workflowpkg.WorkflowLintRequest{Namespace: namespace, Workflow: v},
				)
			}
		case *wfv1.WorkflowEventBinding:
			// noop
		case *wfv1.WorkflowTemplate:
			objName = getObjectName(wf.WorkflowTemplateKind, v, i)
			if opts.ServiceClients.WorkflowTemplatesClient == nil {
				log.Debugf("ignoring %s, not in lint options kinds", objName)
				continue
			}
			res.Linted = true
			if err == nil {
				_, err = opts.ServiceClients.WorkflowTemplatesClient.LintWorkflowTemplate(
					ctx,
					&workflowtemplatepkg.WorkflowTemplateLintRequest{Namespace: namespace, Template: v},
				)
			}
		default:
			continue // silently ignore unknown kinds
		}

		if err != nil {
			res.Errs = append(res.Errs, fmt.Errorf("in %s: %w", objName, err))
		}
	}

	return res
}

func (l *LintResults) Msg() string {
	return l.msg
}

// evaluate must be called before checking the value of l.Success and l.String()
func (l *LintResults) evaluate() *LintResults {
	success := true
	l.anythingLinted = false

	for _, r := range l.Results {
		if !r.Linted {
			continue
		}
		l.anythingLinted = true

		if len(r.Errs) == 0 {
			continue
		}
		success = false
	}

	if !l.anythingLinted {
		success = false
	}

	l.Success = success
	l.msg = l.buildMsg()

	return l
}

func (l *LintResults) buildMsg() string {
	sb := &strings.Builder{}
	for _, r := range l.Results {
		sb.WriteString(l.fmtr.Format(r))
	}

	sb.WriteString(l.fmtr.Summarize(l))

	return sb.String()
}

func getObjectName(kind string, obj metav1.Object, objIndex int) string {
	name := ""
	switch {
	case obj.GetName() != "":
		name = obj.GetName()
	case obj.GetGenerateName() != "":
		name = obj.GetGenerateName()
	default:
		name = fmt.Sprintf("object #%d", objIndex+1)
	}

	return fmt.Sprintf(`"%s" (%s)`, name, kind)
}

func getLintClients(client apiclient.Client, kinds []string) (ServiceClients, error) {
	res := ServiceClients{}
	var err error
	for _, kind := range kinds {
		switch kind {
		case wf.WorkflowPlural, wf.WorkflowShortName:
			res.WorkflowsClient = client.NewWorkflowServiceClient()
		case wf.WorkflowTemplatePlural, wf.WorkflowTemplateShortName:
			res.WorkflowTemplatesClient, err = client.NewWorkflowTemplateServiceClient()
			if err != nil {
				return ServiceClients{}, err
			}

		case wf.CronWorkflowPlural, wf.CronWorkflowShortName:
			res.CronWorkflowsClient, err = client.NewCronWorkflowServiceClient()
			if err != nil {
				return ServiceClients{}, err
			}

		case wf.ClusterWorkflowTemplatePlural, wf.ClusterWorkflowTemplateShortName:
			res.ClusterWorkflowTemplateClient, err = client.NewClusterWorkflowTemplateServiceClient()
			if err != nil {
				return ServiceClients{}, err
			}

		default:
			return res, fmt.Errorf("unknown kind: %s", kind)
		}
	}

	return res, nil
}
