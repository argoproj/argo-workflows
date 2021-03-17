package lint

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterworkflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	wf "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
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
	Format(*LintResults) string
}

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
				path = "stdin"
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

			results.Results = append(results.Results, lintData(ctx, path, data, opts))

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return results.evaluate(), nil
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
			// silently ignore unknown kinds
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

	var fmtr Formatter = defaultFormatter
	if l.fmtr != nil {
		fmtr = l.fmtr
	}
	l.msg = fmtr.Format(l)

	return l
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
