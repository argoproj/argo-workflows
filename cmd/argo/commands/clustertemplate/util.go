package clustertemplate

import (
	"encoding/json"
	"fmt"
	"log"

	argoJson "github.com/argoproj/pkg/json"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/pkg/humanize"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func generateClusterWorkflowTemplates(filePaths []string, strict bool) []wfv1.ClusterWorkflowTemplate {
	fileContents, err := util.ReadManifest(filePaths...)
	if err != nil {
		log.Fatal(err)
	}

	var clusterWorkflowTemplates []wfv1.ClusterWorkflowTemplate
	for _, body := range fileContents {
		cwftmpls, err := unmarshalClusterWorkflowTemplates(body, strict)
		if err != nil {
			log.Fatalf("Failed to parse cluster workflow template: %v", err)
		}
		clusterWorkflowTemplates = append(clusterWorkflowTemplates, cwftmpls...)
	}

	if len(clusterWorkflowTemplates) == 0 {
		log.Fatalln("No cluster workflow template found in given files")
	}

	return clusterWorkflowTemplates
}

// unmarshalClusterWorkflowTemplates unmarshals the input bytes as either json or yaml
func unmarshalClusterWorkflowTemplates(wfBytes []byte, strict bool) ([]wfv1.ClusterWorkflowTemplate, error) {
	var cwft wfv1.ClusterWorkflowTemplate
	var jsonOpts []argoJson.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, argoJson.DisallowUnknownFields)
	}
	err := argoJson.Unmarshal(wfBytes, &cwft, jsonOpts...)
	if err == nil {
		return []wfv1.ClusterWorkflowTemplate{cwft}, nil
	}
	yamlWfs, err := common.SplitClusterWorkflowTemplateYAMLFile(wfBytes, strict)
	if err == nil {
		return yamlWfs, nil
	}
	return nil, err
}

func printClusterWorkflowTemplate(wf *wfv1.ClusterWorkflowTemplate, outFmt string) {
	switch outFmt {
	case "name":
		fmt.Println(wf.ObjectMeta.Name)
	case "json":
		outBytes, _ := json.MarshalIndent(wf, "", "    ")
		fmt.Println(string(outBytes))
	case "yaml":
		outBytes, _ := yaml.Marshal(wf)
		fmt.Print(string(outBytes))
	case "wide", "":
		printClusterWorkflowTemplateHelper(wf)
	default:
		log.Fatalf("Unknown output format: %s", outFmt)
	}
}

func printClusterWorkflowTemplateHelper(wf *wfv1.ClusterWorkflowTemplate) {
	const fmtStr = "%-20s %v\n"
	fmt.Printf(fmtStr, "Name:", wf.ObjectMeta.Name)
	fmt.Printf(fmtStr, "Created:", humanize.Timestamp(wf.ObjectMeta.CreationTimestamp.Time))
}
