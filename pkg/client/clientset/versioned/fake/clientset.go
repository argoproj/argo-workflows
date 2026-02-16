package fake

import "k8s.io/apimachinery/pkg/runtime"

// NewClientset wraps NewSimpleClientset. The upstream k8s fake client
// deprecated NewSimpleClientset in favour of NewClientset (which adds
// field-management support). Argo's code-generator doesn't produce
// apply configurations yet, so the two are identical here.
var NewClientset = func(objects ...runtime.Object) *Clientset {
	return NewSimpleClientset(objects...)
}
