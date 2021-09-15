package casbin

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

type RESTClientEnforced struct {
	delegate rest.Interface
}

func (R RESTClientEnforced) GetRateLimiter() flowcontrol.RateLimiter {
	panic("implement GetRateLimiter")
}

func (R RESTClientEnforced) Verb(verb string) *rest.Request {
	panic("implement Verb")
}

func (R RESTClientEnforced) Post() *rest.Request {
	panic("implement Post")
}

func (R RESTClientEnforced) Put() *rest.Request {
	panic("implement Put")
}

func (R RESTClientEnforced) Patch(pt types.PatchType) *rest.Request {
	panic("implement Patch")
}

func (R RESTClientEnforced) Get() *rest.Request {
	panic("implement Get")
}

func (R RESTClientEnforced) Delete() *rest.Request {
	panic("implement Delete")
}

func (R RESTClientEnforced) APIVersion() schema.GroupVersion {
	panic("implement APIVersion")
}

func (c WorkflowEnforcedInterface) RESTClient() rest.Interface {
	return &RESTClientEnforced{c.delegate.ArgoprojV1alpha1().RESTClient()}
}
