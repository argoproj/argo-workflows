package main

import (
	"fmt"
	"log"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	plugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/controller"
)

type plugin struct{}

func New(map[string]string) (interface{}, error) { //nolint:deadcode
	log.Println("Hello! Just starting up...")
	return &plugin{}, nil
}

func main() {
	// main funcs are never called in a Go plugin
}

var _ plugins.WorkflowLifecycleHook = &plugin{}

func (p *plugin) WorkflowPreOperate(args plugins.WorkflowPreOperateArgs, reply *plugins.WorkflowPreOperateReply) error {
	if _, ok := args.Workflow.Annotations["hello"]; ok && args.Workflow.Status.Phase == wfv1.WorkflowUnknown {
		log.Println("hello: setting hello annotation to running")
		reply.Workflow = args.Workflow
		reply.Workflow.Annotations["hello"] = "running"
	}
	return nil
}

func (p *plugin) WorkflowPostOperate(args plugins.WorkflowPostOperateArgs, reply *plugins.WorkflowPostOperateReply) error {
	if args.New.Annotations["hello"] == "running" {
		log.Println("hello: updating hello annotation")
		reply.New = args.New
		reply.New.Annotations["hello"] = "goodbye"
	}
	return nil
}

var _ plugins.NodeLifecycleHook = &plugin{}

func (p *plugin) NodePreExecute(args plugins.NodePreExecuteArgs, reply *plugins.NodePreExecuteReply) error {
	value, err := args.Template.Plugin.Get("helloController")
	if err != nil {
		return err
	}
	if value != nil {
		log.Printf("hello: executing hello plugin node: %v\n", value)
		reply.Node = args.Node
		reply.Node.Phase = wfv1.NodeSucceeded
		reply.Node.Message = fmt.Sprintf("Hello %s: %s", args.Workflow.Name, reply.Node.ID)
	}
	return nil
}

func (p *plugin) NodePostExecute(args plugins.NodePostExecuteArgs, reply *plugins.NodePostExecuteReply) error {
	value, err := args.Template.Plugin.Get("helloController")
	if err != nil {
		return err
	}
	if value != nil {
		log.Printf("executor hello plugin node: %v\n", value)
	}
	return nil
}

var _ plugins.PodLifecycleHook = plugin{}

func (p plugin) PodPreCreate(args plugins.PodPreCreateArgs, reply *plugins.PodPreCreateReply) error {
	if _, ok := args.Workflow.Annotations["hello"]; ok {
		log.Printf("hello: annotating pod: %s\n", args.Pod.Name)
		reply.Pod = args.Pod
		reply.Pod.Annotations["hello"] = "here we are!"
	}
	return nil
}

func (p plugin) PodPostCreate(args plugins.PodPostCreateArgs, reply *plugins.PodPostCreateReply) error {
	if _, ok := args.Workflow.Annotations["hello"]; ok {
		log.Printf("hello: created pod: %s\n", args.Pod.Name)
	}
	return nil
}

var _ plugins.ParameterSubstitutionPlugin = &plugin{}

func (p *plugin) ParameterPreSubstitution(args plugins.ParameterPreSubstitutionArgs, reply *plugins.ParameterPreSubstitutionReply) error {
	reply.Parameters = map[string]string{}
	reply.Parameters["hello"] = "good morning"
	return nil
}
