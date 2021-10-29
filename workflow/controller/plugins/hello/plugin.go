package main

import (
	"fmt"
	"log"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/plugins"
)

type plugin struct{}

var _ plugins.WorkflowLifecycleHook = plugin{}
var _ plugins.TemplateExecutor = plugin{}

func init() {
	log.Println("Hello! Just starting up...")
}

func (plugin) WorkflowPreOperate(args plugins.WorkflowPreOperateArgs, _ *plugins.WorkflowPreOperateReply) error { //nolint:unparam
	if _, ok := args.Workflow.Annotations["hello"]; ok && args.Workflow.Status.Phase == wfv1.WorkflowUnknown {
		log.Println("setting hello to running")
		args.Workflow.Annotations["hello"] = "running"
	}
	return nil
}

func (plugin) WorkflowPreUpdate(args plugins.WorkflowPreUpdateArgs, _ *plugins.WorkflowPreUpdateReply) error { //nolint:unparam
	if args.New.Annotations["hello"] == "running" {
		log.Println("setting hello to goodbye")
		args.New.Annotations["hello"] = "goodbye"
	}
	return nil
}

func (plugin) ExecuteTemplate(args plugins.ExecuteTemplateArgs, reply *plugins.ExecuteTemplateReply) error { //nolint:unparam
	if args.Template.Plugin == nil {
		// we could execute other types too
		return nil
	}
	value, err := args.Template.Plugin.AsMap()
	if err != nil {
		return err
	}
	log.Printf("executing hello plugin template: %v", value)
	if _, ok := value["hello"]; ok {
		reply.Node.Phase = wfv1.NodeSucceeded
		reply.Node.Message = fmt.Sprintf("Hello %s: %s", args.Workflow.Name, reply.Node.ID)
	}
	return nil
}

var Plugin = plugin{} //nolint:deadcode
