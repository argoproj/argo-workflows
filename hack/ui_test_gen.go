package main

import (
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util/exec"
	"github.com/sirupsen/logrus"
)

func Gen(workflowName string, args ...string) {
	wfRaw, stderr, err := exec.ExecSplit("kubectl", "get", "wf", workflowName, "-o", "json")
	if stderr != "" || err != nil {
		logrus.Fatalf("%s: %s", err, stderr)
	}

	var wf v1alpha1.Workflow
	err = json.Unmarshal([]byte(wfRaw), &wf)
	if err != nil {
		logrus.Fatal(err)
	}

	out := ""
	if shouldGen(args, "workflowName") {
		out += fmt.Sprintf(`workflowName={'%s'} `, wf.Name)
	}
	if shouldGen(args, "nodes") {
		nodes, err := json.MarshalIndent(wf.Status.Nodes, "", "  ")
		if err != nil {
			logrus.Fatal(err)
		}
		out += fmt.Sprintf(`nodes={%s} `, nodes)
	}
	fmt.Println(out)
}

func shouldGen(args []string, target string) bool {
	for _, item := range args {
		if target == item {
			return true
		}
	}
	return false
}
