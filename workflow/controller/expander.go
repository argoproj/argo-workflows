package controller

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/argoproj/argo/errors"
	"github.com/valyala/fasttemplate"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type ExpandableCollector func(template *fasttemplate.Template, expandable wfv1.Expandable, i int, item wfv1.Item) error

func getStepCollector(expandedSteps *[]wfv1.WorkflowStep) ExpandableCollector {
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

func getTaskCollector(expandedTasks *[]wfv1.DAGTask) ExpandableCollector {
	return func(template *fasttemplate.Template, expandable wfv1.Expandable, i int, item wfv1.Item) error {
		var newStep wfv1.DAGTask
		newName, err := processItem(template, expandable.GetName(), i, item, &newStep)
		if err != nil {
			return err
		}
		newStep.Name = newName
		newStep.Template = expandable.GetTemplate()
		*expandedTasks = append(*expandedTasks, newStep)
		return nil
	}
}

// expandStep expands a step containing withItems or withParams into multiple parallel steps
func expand(expandable wfv1.Expandable, collector ExpandableCollector) error {
	var err error
	var items []wfv1.Item
	if len(expandable.GetWithItems()) > 0 {
		items = expandable.GetWithItems()
	} else if expandable.GetWithParam() != "" {
		err = json.Unmarshal([]byte(expandable.GetWithParam()), &items)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "withParam value could not be parsed as a JSON list: %s", strings.TrimSpace(expandable.GetWithParam()))
		}
	} else if expandable.GetWithSequence() != nil {
		items, err = expandSequence(expandable.GetWithSequence())
		if err != nil {
			return err
		}
	} else {
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
