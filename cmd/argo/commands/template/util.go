package template

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/humanize"
	argoJson "github.com/argoproj/argo-workflows/v3/util/json"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func generateWorkflowTemplates(ctx context.Context, filePaths []string, strict bool) []wfv1.WorkflowTemplate {
	fileContents, err := util.ReadManifest(filePaths...)
	if err != nil {
		log.Fatal(err)
	}

	var workflowTemplates []wfv1.WorkflowTemplate
	for _, body := range fileContents {
		wftmpls := unmarshalWorkflowTemplates(ctx, body, strict)
		workflowTemplates = append(workflowTemplates, wftmpls...)
	}

	if len(workflowTemplates) == 0 {
		log.Fatalln("No workflow template found in given files")
	}

	return workflowTemplates
}

// unmarshalWorkflowTemplates unmarshals the input bytes as either json or yaml
func unmarshalWorkflowTemplates(ctx context.Context, wfBytes []byte, strict bool) []wfv1.WorkflowTemplate {
	var wf wfv1.WorkflowTemplate
	var jsonOpts []argoJson.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, argoJson.DisallowUnknownFields)
	}
	err := argoJson.Unmarshal(wfBytes, &wf, jsonOpts...)
	if err == nil {
		return []wfv1.WorkflowTemplate{wf}
	}
	yamlWfs, err := common.SplitWorkflowTemplateYAMLFile(ctx, wfBytes, strict)
	if err == nil {
		return yamlWfs
	}
	log.Fatalf("Failed to parse workflow template: %v", err)
	return nil
}

func printWorkflowTemplate(wf *wfv1.WorkflowTemplate, outFmt string) {
	switch outFmt {
	case "name":
		fmt.Println(wf.Name)
	case "json":
		outBytes, _ := json.MarshalIndent(wf, "", "    ")
		fmt.Println(string(outBytes))
	case "yaml":
		outBytes, _ := yaml.Marshal(wf)
		fmt.Print(string(outBytes))
	case "wide", "":
		printWorkflowTemplateHelper(wf)
	default:
		log.Fatalf("Unknown output format: %s", outFmt)
	}
}

func printWorkflowTemplateHelper(wf *wfv1.WorkflowTemplate) {
	const fmtStr = "%-20s %v\n"
	fmt.Printf(fmtStr, "Name:", wf.Name)
	fmt.Printf(fmtStr, "Namespace:", wf.Namespace)
	fmt.Printf(fmtStr, "Created:", humanize.Timestamp(wf.CreationTimestamp.Time))
}
