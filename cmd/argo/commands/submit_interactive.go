package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"google.golang.org/grpc/status"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient"
	clusterworkflowtmplpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// addInteractiveParameters asks for the parameters of the resource that --from points at, and adds the
// answers to submitOpts.Parameters.
func addInteractiveParameters(ctx context.Context, apiClient apiclient.Client, serviceClient workflowpkg.WorkflowServiceClient, namespace, from string, submitOpts *wfv1.SubmitOpts, in io.Reader, out io.Writer) error {
	kind, name, err := parseResourceIdentifier(from)
	if err != nil {
		return err
	}
	params, err := getParametersForInteractiveSubmit(ctx, apiClient, namespace, kind, name)
	if err != nil {
		return err
	}
	answers, err := promptForParameters(ctx, in, out, params, submitOpts.Parameters, newEnumResolverFunc(serviceClient, namespace, kind, name))
	if err != nil {
		return err
	}
	submitOpts.Parameters = append(submitOpts.Parameters, answers...)
	return nil
}

// getParametersForInteractiveSubmit fetches the workflow parameters declared by the resource that --from points at.
func getParametersForInteractiveSubmit(ctx context.Context, apiClient apiclient.Client, namespace, kind, name string) ([]wfv1.Parameter, error) {
	switch kind {
	case workflow.WorkflowTemplateKind, workflow.WorkflowTemplateSingular, workflow.WorkflowTemplatePlural, workflow.WorkflowTemplateShortName:
		c, err := apiClient.NewWorkflowTemplateServiceClient()
		if err != nil {
			return nil, err
		}
		tmpl, err := c.GetWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateGetRequest{Name: name, Namespace: namespace})
		if err != nil {
			return nil, err
		}
		return tmpl.Spec.Arguments.Parameters, nil
	case workflow.ClusterWorkflowTemplateKind, workflow.ClusterWorkflowTemplateSingular, workflow.ClusterWorkflowTemplatePlural, workflow.ClusterWorkflowTemplateShortName:
		c, err := apiClient.NewClusterWorkflowTemplateServiceClient()
		if err != nil {
			return nil, err
		}
		tmpl, err := c.GetClusterWorkflowTemplate(ctx, &clusterworkflowtmplpkg.ClusterWorkflowTemplateGetRequest{Name: name})
		if err != nil {
			return nil, err
		}
		return tmpl.Spec.Arguments.Parameters, nil
	case workflow.CronWorkflowKind, workflow.CronWorkflowSingular, workflow.CronWorkflowPlural, workflow.CronWorkflowShortName:
		c, err := apiClient.NewCronWorkflowServiceClient()
		if err != nil {
			return nil, err
		}
		cronWf, err := c.GetCronWorkflow(ctx, &cronworkflowpkg.GetCronWorkflowRequest{Name: name, Namespace: namespace})
		if err != nil {
			return nil, err
		}
		return cronWf.Spec.WorkflowSpec.Arguments.Parameters, nil
	default:
		return nil, fmt.Errorf("--interactive does not support resource kind %q", kind)
	}
}

// enumResolverFunc returns the options for a parameter with enumFrom, given the current parameter values.
type enumResolverFunc func(ctx context.Context, parameterName string, values map[string]string) (*workflowpkg.ResolveParameterEnumResponse, error)

func newEnumResolverFunc(serviceClient workflowpkg.WorkflowServiceClient, namespace, kind, name string) enumResolverFunc {
	return func(ctx context.Context, parameterName string, values map[string]string) (*workflowpkg.ResolveParameterEnumResponse, error) {
		return serviceClient.ResolveParameterEnum(ctx, &workflowpkg.ResolveParameterEnumRequest{
			Namespace:     namespace,
			ResourceKind:  kind,
			ResourceName:  name,
			ParameterName: parameterName,
			Parameters:    values,
		})
	}
}

// promptForParameters asks for every parameter not already given as name=value in `given`, in declaration
// order, and returns the answers as name=value. Parameters with enum or enumFrom are chosen from a numbered
// list; if enumFrom can't be resolved, the user types the value instead. Answers feed into later enumFrom
// lookups, so a parameter can depend on one declared before it.
func promptForParameters(ctx context.Context, in io.Reader, out io.Writer, params []wfv1.Parameter, given []string, resolve enumResolverFunc) ([]string, error) {
	values := map[string]string{}
	for _, p := range params {
		if v := parameterDefault(p); v != nil {
			values[p.Name] = *v
		}
	}
	isGiven := map[string]bool{}
	for _, kv := range given {
		name, value, _ := strings.Cut(kv, "=")
		values[name] = value
		isGiven[name] = true
	}

	reader := bufio.NewReader(in)
	var answers []string
	for _, p := range params {
		if isGiven[p.Name] {
			continue
		}
		fmt.Fprintf(out, "\n%s", p.Name)
		if p.Description != nil && p.Description.String() != "" {
			fmt.Fprintf(out, " - %s", p.Description.String())
		}
		fmt.Fprintln(out)

		var options []string
		switch {
		case p.EnumFrom != nil:
			resp, err := resolve(ctx, p.Name, values)
			switch {
			case err != nil:
				// status.Convert drops the "rpc error: code = ... desc =" prefix; plain errors pass through.
				fmt.Fprintf(out, "  could not load options, type a value instead: %s\n", status.Convert(err).Message())
			case len(resp.Values) == 0:
				fmt.Fprintln(out, "  no options found, type a value instead")
			default:
				options = resp.Values
				if resp.Truncated {
					fmt.Fprintf(out, "  showing the first %d options only\n", len(options))
				}
			}
		case len(p.Enum) > 0:
			for _, e := range p.Enum {
				options = append(options, e.String())
			}
		}

		def := parameterDefault(p)
		value, err := promptForValue(reader, out, options, def)
		if err != nil {
			return nil, fmt.Errorf("parameter %q: %w", p.Name, err)
		}
		values[p.Name] = value
		answers = append(answers, p.Name+"="+value)
	}
	return answers, nil
}

// promptForValue reads one answer. With options, the answer must be an option's number or the option itself;
// an empty answer picks the default. Without options, any text is accepted.
func promptForValue(reader *bufio.Reader, out io.Writer, options []string, def *string) (string, error) {
	if def != nil && *def == "" && len(options) > 0 {
		def = nil // an empty placeholder is not a real choice
	}
	for i, o := range options {
		fmt.Fprintf(out, "  %d) %s\n", i+1, o)
	}
	for {
		switch {
		case len(options) > 0 && def != nil:
			fmt.Fprintf(out, "  choose 1-%d [%s]: ", len(options), *def)
		case len(options) > 0:
			fmt.Fprintf(out, "  choose 1-%d: ", len(options))
		case def != nil:
			fmt.Fprintf(out, "  value [%s]: ", *def)
		default:
			fmt.Fprint(out, "  value: ")
		}
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		eof := errors.Is(err, io.EOF)
		answer := strings.TrimSpace(line)

		if answer == "" {
			if def != nil {
				return *def, nil
			}
			if eof {
				return "", errors.New("no value given")
			}
			if len(options) == 0 {
				return "", nil
			}
		} else {
			if len(options) == 0 {
				return answer, nil
			}
			if v, ok := matchOption(options, answer); ok {
				return v, nil
			}
		}
		if eof {
			return "", fmt.Errorf("%q is not one of the options", answer)
		}
		fmt.Fprintf(out, "  enter a number from 1 to %d, or one of the options\n", len(options))
	}
}

// matchOption accepts an option's number, or the option itself. A number wins over an option that looks like one.
func matchOption(options []string, answer string) (string, bool) {
	if n, err := strconv.Atoi(answer); err == nil && n >= 1 && n <= len(options) {
		return options[n-1], true
	}
	if slices.Contains(options, answer) {
		return answer, true
	}
	return "", false
}

// parameterDefault is the value the UI pre-fills: value if set, otherwise default.
func parameterDefault(p wfv1.Parameter) *string {
	if p.Value != nil {
		v := p.Value.String()
		return &v
	}
	if p.Default != nil {
		v := p.Default.String()
		return &v
	}
	return nil
}
