package controller

import (
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo/errors"
	"github.com/valyala/fasttemplate"
	"strings"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type expandableCollector func(template *fasttemplate.Template, expandable wfv1.Expandable, i int, item wfv1.Item) error

func getStepCollector(expandedSteps *[]wfv1.WorkflowStep) expandableCollector {
	return func(template *fasttemplate.Template, expandable wfv1.Expandable, i int, item wfv1.Item) error {
		var newStep wfv1.WorkflowStep
		newName, err := processItem(template, expandable.GetName(), i, item, &newStep)
		if err != nil {
			return err
		}
		newStep.Name = newName
		newStep.Template = expandable.GetTemplate()
		*expandedSteps = append(*expandedSteps, newStep)
		return nil
	}
}

func getTaskCollector(expandedTasks *[]wfv1.DAGTask) expandableCollector {
	return func(template *fasttemplate.Template, expandable wfv1.Expandable, i int, item wfv1.Item) error {
		var newTask wfv1.DAGTask
		newName, err := processItem(template, expandable.GetName(), i, item, &newTask)
		if err != nil {
			return err
		}
		newTask.Name = newName
		newTask.Template = expandable.GetTemplate()
		*expandedTasks = append(*expandedTasks, newTask)
		return nil
	}
}

// expandStep expands a step containing withItems or withParams into multiple parallel steps
func expand(expandable wfv1.Expandable, collector expandableCollector) error {
	var err error
	var items []wfv1.Item

	switch {
	case len(expandable.GetWithItems()) > 0:
		items = expandable.GetWithItems()

	case expandable.GetWithParam() != "":
		err = json.Unmarshal([]byte(expandable.GetWithParam()), &items)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "withParam value could not be parsed as a JSON list: %s", strings.TrimSpace(expandable.GetWithParam()))
		}

	case expandable.GetWithSequence() != nil:
		items, err = expandSequence(expandable.GetWithSequence())
		if err != nil {
			return err
		}

	default:
		// this should have been prevented in expandStepGroup()
		return errors.InternalError("expandStep() was called with withItems and withParam empty")
	}

	// these fields can be very large (>100m) and marshalling 10k x 100m = 6GB of memory used and
	// very poor performance, so we just nil them out
	expandable.NilFields()

	stepBytes, err := json.Marshal(expandable)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	fstTmpl, err := fasttemplate.NewTemplate(string(stepBytes), "{{", "}}")
	if err != nil {
		return fmt.Errorf("unable to parse argo varaible: %w", err)
	}

	for i, item := range items {
		if err := collector(fstTmpl, expandable, i, item); err != nil {
			return err
		}
	}
	return nil
}
