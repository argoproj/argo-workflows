package casbin

import (
	"context"
	"fmt"
	"os"

	"github.com/casbin/casbin/v2"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workflow "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	authclaims "github.com/argoproj/argo-workflows/v3/server/auth/types"
)

type Sub struct { // TODO - own file
	Sub string
}

func (s Sub) String() string { // TODO - tests
	return s.Sub
}

type Obj struct { // TODO - own file
	Resource  string
	Namespace string
	Name      string
}

func (o Obj) String() string { // TODO - tests
	return fmt.Sprintf("%s/%s/%s", o.Resource, o.Namespace, o.Name)
}

var enforce = func(ctx context.Context, resource, namespace, name, verb string) error { return nil }
var GetClaims func(ctx context.Context) *authclaims.Claims

func init() {
	println("ALEX", "init") // TODO - replace with debug logging
	// these files must be mounted at /casbin using configmap volume mount
	// TODO if these files are not found then log they are not found,
	// and then use a "allow everything" enforcer
	e, err := casbin.NewEnforcer("casbin/model.conf", "casbin/policy.csv")
	if os.IsNotExist(err) {
		log.WithError(err).Info("Casbin RBAC disabled")
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	e.EnableLog(true) // TODO - do we want config for this? e.g. ARGO_CASBIN_ENABLE_LOG=true
	enforce = func(ctx context.Context, resource, namespace, name, verb string) error {
		claims := GetClaims(ctx)
		sub := Sub{Sub: "anonymous"} // TODO -is this the best name for an "anonymous" users? could it conflict with a real user with name "anonymous"
		if claims != nil {
			sub.Sub = claims.Subject // TODO - needs testing
		}
		obj := Obj{Resource: resource, Namespace: namespace, Name: name}
		act := verb
		// TODO - claims exposes
		// - email - because many "subjects" are opaque strings - do we want to use that somehow?
		// - groups - can we map these to roles somehow?
		println("ALEX", "enforce", sub.String(), obj.String(), act) // TODO - replace with debug logging

		if ok, err := e.Enforce(sub, obj, act); err != nil {
			return err
		} else if !ok {
			return status.Error(codes.Unauthenticated, "not allowed")
		}
		return nil
	}
}

type foo struct { // TODO - new name and own file
	x workflow.Interface // TODO - rename to "delegate"
}

func (c foo) RESTClient() rest.Interface {
	panic("not supported") // not all of these need to be implemented, this one is unused for example
}

func (c foo) ClusterWorkflowTemplates() v1alpha1.ClusterWorkflowTemplateInterface {
	panic("implement me")
}

func (c foo) CronWorkflows(namespace string) v1alpha1.CronWorkflowInterface {
	panic("implement me")
}

type bar struct { // TODO - new name and own file
	x         v1alpha1.WorkflowInterface // TODO - rename to "delegate"
	namespace string
}

func (c bar) Create(ctx context.Context, workflow *wfv1.Workflow, opts metav1.CreateOptions) (*wfv1.Workflow, error) {
	panic("implement me")
}

func (c bar) Update(ctx context.Context, workflow *wfv1.Workflow, opts metav1.UpdateOptions) (*wfv1.Workflow, error) {
	panic("implement me")
}

func (c bar) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	panic("implement me")
}

func (c bar) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("implement me")
}

func (c bar) Get(ctx context.Context, name string, opts metav1.GetOptions) (*wfv1.Workflow, error) {
	panic("implement me")
}

func (c bar) List(ctx context.Context, opts metav1.ListOptions) (*wfv1.WorkflowList, error) {
	if err := enforce(ctx, "workflows", c.namespace, "", "list"); err != nil {
		return nil, err
	}
	return c.x.List(ctx, opts)
}

func (c bar) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (c bar) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *wfv1.Workflow, err error) {
	panic("implement me")
}

func (c foo) Workflows(namespace string) v1alpha1.WorkflowInterface {
	println("ALEX", "Workflows", namespace)
	return &bar{c.x.ArgoprojV1alpha1().Workflows(namespace), namespace}
}

func (c foo) WorkflowEventBindings(namespace string) v1alpha1.WorkflowEventBindingInterface {
	panic("implement me")
}

func (c foo) WorkflowTaskSets(namespace string) v1alpha1.WorkflowTaskSetInterface {
	panic("implement me")
}

func (c foo) WorkflowTemplates(namespace string) v1alpha1.WorkflowTemplateInterface {
	panic("implement me")
}

func (c foo) Discovery() discovery.DiscoveryInterface {
	panic("not supported")
}

func (c foo) ArgoprojV1alpha1() v1alpha1.ArgoprojV1alpha1Interface {
	return c
}

func WrapWorkflowInterface(x workflow.Interface) workflow.Interface {
	println("ALEX", "workflowInterface")
	return &foo{x}
}
