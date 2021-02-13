package lint

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	clusterworkflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type (
	ServiceClients struct {
		WorkflowsClient               workflowpkg.WorkflowServiceClient
		WorkflowTemplatesClient       workflowtemplatepkg.WorkflowTemplateServiceClient
		CronWorkflowsClient           cronworkflowpkg.CronWorkflowServiceClient
		ClusterWorkflowTemplateClient clusterworkflowtemplatepkg.ClusterWorkflowTemplateServiceClient
	}

	LintOptions struct {
		Files            []string
		Strict           bool
		DefaultNamespace string
		Formatter        Formatter
		ServiceClients   ServiceClients
	}

	LintResult struct {
		File   string
		Errs   []error
		Linted bool
	}

	LintResults struct {
		Results        []*LintResult
		Success        bool
		msg            string
		fmtr           Formatter
		anythingLinted bool
	}

	Formatter interface {
		Format(*LintResults) string
	}
)

var (
	lintExt = map[string]bool{
		".yaml": true,
		".yml":  true,
		".json": true,
	}

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

// Lint reads all files, returns linting errors of all of the enitities of the specified kinds.
// Entities of other kinds are ignored.
func Lint(ctx context.Context, opts *LintOptions) (*LintResults, error) {
	results := &LintResults{
		Results: []*LintResult{},
		fmtr:    opts.Formatter,
	}

	for _, file := range opts.Files {
		err := filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
			var r io.Reader
			switch {
			case path == "-":
				r = os.Stdin
			case err != nil:
				return err
			case lintExt[filepath.Ext(path)]:
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				r = f
			case info.IsDir():
				return nil // skip
			default:
				log.Warnf("ignoring file with unknown extension: %s", path)
				return nil
			}

			data, err := ioutil.ReadAll(r)
			if err != nil {
				return err
			}

			lintRes, err := lintData(ctx, path, data, opts)
			if err != nil {
				return err
			}
			results.Results = append(results.Results, lintRes)

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return results.evaluate(), nil
}

func lintData(ctx context.Context, src string, data []byte, opts *LintOptions) (*LintResult, error) {
	if src == "-" {
		src = "stdin"
	}
	res := &LintResult{
		File: src,
		Errs: []error{},
	}
	objects, err := common.ParseObjects(data, opts.Strict)
	if err != nil {
		res.Linted = true
		res.Errs = append(res.Errs, fmt.Errorf("failed to parse objects from %s: %s", src, err))
		return res, nil
	}

	for i, obj := range objects {
		var err error // shadow above
		// we should prefer the object's namespace
		namespace := obj.GetNamespace()
		if namespace == "" {
			namespace = opts.DefaultNamespace
		}

		switch v := obj.(type) {
		case *wfv1.ClusterWorkflowTemplate:
			if opts.ServiceClients.ClusterWorkflowTemplateClient == nil {
				log.Debug("ignoring kind: ClusterWorkflowTemplate, not in lint options")
				continue
			}
			res.Linted = true
			_, err = opts.ServiceClients.ClusterWorkflowTemplateClient.LintClusterWorkflowTemplate(
				ctx,
				&clusterworkflowtemplatepkg.ClusterWorkflowTemplateLintRequest{Template: v},
			)
		case *wfv1.CronWorkflow:
			if opts.ServiceClients.CronWorkflowsClient == nil {
				log.Debug("ignoring kind: CronWorkflow, not in lint options kinds")
				continue
			}
			res.Linted = true
			_, err = opts.ServiceClients.CronWorkflowsClient.LintCronWorkflow(
				ctx,
				&cronworkflowpkg.LintCronWorkflowRequest{Namespace: namespace, CronWorkflow: v},
			)
		case *wfv1.Workflow:
			if opts.ServiceClients.WorkflowsClient == nil {
				log.Debug("ignoring kind: Workflow, not in lint options kinds")
				continue
			}
			res.Linted = true
			_, err = opts.ServiceClients.WorkflowsClient.LintWorkflow(
				ctx,
				&workflowpkg.WorkflowLintRequest{Namespace: namespace, Workflow: v},
			)
		case *wfv1.WorkflowEventBinding:
			// noop
		case *wfv1.WorkflowTemplate:
			if opts.ServiceClients.WorkflowTemplatesClient == nil {
				log.Debug("ignoring kind: WorkflowTemplate, not in lint options kinds")
				continue
			}
			res.Linted = true
			_, err = opts.ServiceClients.WorkflowTemplatesClient.LintWorkflowTemplate(
				ctx,
				&workflowtemplatepkg.WorkflowTemplateLintRequest{Namespace: namespace, Template: v},
			)
		default:
			// silently ignore unknown kinds
		}

		if err != nil {
			res.Errs = append(res.Errs, fmt.Errorf("in object #%d: %s", i+1, err))
		}
	}

	return res, nil
}

func (l *LintResults) String() string {
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

	var fmtr Formatter = defaultFormatter
	if l.fmtr != nil {
		fmtr = l.fmtr
	}
	l.msg = fmtr.Format(l)

	return l
}
