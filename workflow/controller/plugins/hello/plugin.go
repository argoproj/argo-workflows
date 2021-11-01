package main

import (
	"fmt"
	"log"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/plugins"
)

type plugin struct{}

var Plugin = plugin{} //nolint:deadcode

func init() {
	log.Println("Hello! Just starting up...")
}

func main() {
	// main funcs are never called in a Go plugin
}

var _ plugins.WorkflowLifecycleHook = plugin{}

func (plugin) WorkflowPreOperate(args plugins.WorkflowPreOperateArgs, reply *plugins.WorkflowPreOperateReply) error { //nolint:unparam
	if _, ok := reply.Workflow.Annotations["hello"]; ok && reply.Workflow.Status.Phase == wfv1.WorkflowUnknown {
		log.Println("hello: setting hello annotation to running")
		reply.Workflow.Annotations["hello"] = "running"
	}
	return nil
}

func (plugin) WorkflowPreUpdate(args plugins.WorkflowPreUpdateArgs, reply *plugins.WorkflowPreUpdateReply) error { //nolint:unparam
	if reply.New.Annotations["hello"] == "running" {
		log.Println("hello: updating hello annotation")
		reply.New.Annotations["hello"] = "goodbye"
	}
	return nil
}

var _ plugins.NodeLifecycleHook = plugin{}

func (plugin) NodePreExecute(args plugins.NodePreExecuteArgs, reply *plugins.NodePreExecuteReply) error { //nolint:unparam
	value, err := args.Template.Plugin.Get("hello")
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

func (plugin) NodePostExecute(args plugins.NodePostExecuteArgs, reply *plugins.NodePostExecuteReply) error {
	value, err := args.Template.Plugin.Get("hello")
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
		log.Printf("hello: annotating pod: %s\n", reply.Pod.Name)
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

var _ plugins.ParameterSubstitutionPlugin = plugin{}

func (plugin) ParameterPreSubstitution(args plugins.ParameterPreSubstitutionArgs, reply *plugins.ParameterPreSubstitutionReply) error {
	if _, ok := args.Workflow.Annotations["hello"]; ok {
		log.Printf("hello: adding hello parameter: %s\n", args.Workflow.Name)
		reply.Parameters = map[string]string{}
		reply.Parameters["hello"] = "good morning"
	}
	return nil
}
