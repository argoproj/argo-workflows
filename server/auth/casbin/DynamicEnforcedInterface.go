package casbin

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

type DynamicEnforcedInterface struct {
	delegate dynamic.Interface
}

func (d DynamicEnforcedInterface) Namespace(s string) dynamic.ResourceInterface {
	return d
}

func (d DynamicEnforcedInterface) Create(ctx context.Context, obj *unstructured.Unstructured, options metav1.CreateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	panic("implement me")
}

func (d DynamicEnforcedInterface) Update(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	panic("implement me")
}

func (d DynamicEnforcedInterface) UpdateStatus(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	panic("implement me")
}

func (d DynamicEnforcedInterface) Delete(ctx context.Context, name string, options metav1.DeleteOptions, subresources ...string) error {
	panic("implement me")
}

func (d DynamicEnforcedInterface) DeleteCollection(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	panic("implement me")
}

func (d DynamicEnforcedInterface) Get(ctx context.Context, name string, options metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
	panic("implement me")
}

func (d DynamicEnforcedInterface) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	panic("implement me")
}

func (d DynamicEnforcedInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (d DynamicEnforcedInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*unstructured.Unstructured, error) {
	panic("implement me")
}

func (d DynamicEnforcedInterface) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	return d
}

func WrapDynamicInterface(d dynamic.Interface) dynamic.Interface {
	return &DynamicEnforcedInterface{ d }
}