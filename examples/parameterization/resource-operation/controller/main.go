// Reference adapter for the isolated HTTP operation protocol demo. This is an
// optional operator, not part of the Argo scheduling controller. Its endpoint is
// operator-configured and cannot be supplied through workflow parameter values.
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"time"

	api "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
	op "github.com/argoproj/argo-workflows/v4/workflow/resourceoperation"
	"github.com/xeipuuv/gojsonschema"
	core "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"
)

var operations = schema.GroupVersionResource{Group: api.GroupName, Version: "v1alpha1", Resource: "resourceoperations"}
var workflows = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "workflows"}

type adapter struct {
	d                      dynamic.Interface
	k                      kubernetes.Interface
	ns, endpoint, instance string
	client                 *http.Client
}

func (a *adapter) SaveStatus(ctx context.Context, o *api.ResourceOperation) error {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
	if err != nil {
		return err
	}
	saved, err := a.d.Resource(operations).Namespace(a.ns).UpdateStatus(ctx, &unstructured.Unstructured{Object: m}, metav1.UpdateOptions{})
	if err == nil {
		o.ResourceVersion = saved.GetResourceVersion()
	}
	return err
}
func (a *adapter) Verify(ctx context.Context, o *api.ResourceOperation) (bool, error) {
	s := o.Spec.Operation
	if o.Namespace != a.ns || s.Target.ConnectionRef != a.instance || s.Target.ResourceType != "report" {
		return false, errors.New("unsupported demo target")
	}
	if s.IdempotencyKey != s.WorkflowUID+":"+s.NodeID {
		return false, errors.New("operation key is not the stable invocation identity")
	}
	if o.Status.RequestDigest == "" {
		pod, err := a.k.CoreV1().Pods(a.ns).Get(ctx, o.Annotations[api.GroupName+"/invoker-pod"], metav1.GetOptions{})
		if err != nil {
			return false, errors.New("invoking pod not available")
		}
		owned := false
		for _, ref := range pod.OwnerReferences {
			if ref.Kind == "Workflow" && string(ref.UID) == s.WorkflowUID {
				owned = true
			}
		}
		if !owned || pod.Status.Phase != core.PodRunning || pod.Spec.ServiceAccountName != "workflow-executor" || pod.Annotations["workflows.argoproj.io/node-name"] != s.NodeID {
			return false, errors.New("invoking pod does not match active native task")
		}
		verified := false
		for _, c := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
			for _, e := range c.Env {
				if e.Name != "ARGO_TEMPLATE" {
					continue
				}
				var t wfv1.Template
				if json.Unmarshal([]byte(e.Value), &t) != nil || t.Resource == nil {
					continue
				}
				var manifest api.ResourceOperation
				if yaml.Unmarshal([]byte(t.Resource.Manifest), &manifest) != nil {
					continue
				}
				// Control can be independently updated after native creation. All external
				// execution inputs must match the materialized, controller-created Pod.
				if manifest.Name == o.Name && reflect.DeepEqual(manifest.Spec.Operation, s) {
					verified = true
				}
			}
		}
		if !verified {
			return false, errors.New("operation does not match resolved native resource manifest")
		}
	}
	w, err := a.d.Resource(workflows).Namespace(a.ns).Get(ctx, o.Annotations[api.GroupName+"/workflow"], metav1.GetOptions{})
	if kerrors.IsNotFound(err) && o.Status.RequestDigest != "" {
		return true, nil
	}
	if err != nil {
		return false, errors.New("workflow identity unavailable")
	}
	if string(w.GetUID()) != s.WorkflowUID {
		return false, errors.New("workflow identity replaced")
	}
	if o.Status.RequestDigest == "" {
		ref := s.ActionRef
		receiptName := w.GetAnnotations()[api.GroupName+"/submission"]
		receipt, e := a.d.Resource(schema.GroupVersionResource{Group: api.GroupName, Version: "v1alpha1", Resource: "workflowsubmissions"}).Namespace(a.ns).Get(ctx, receiptName, metav1.GetOptions{})
		if e != nil {
			return false, errors.New("frozen submission unavailable")
		}
		if err := validateFrozenAction(ref, receipt); err != nil {
			return false, err
		}
	}
	shutdown, _, _ := unstructured.NestedString(w.Object, "spec", "shutdown")
	phase, _, _ := unstructured.NestedString(w.Object, "status", "phase")
	// Suspend deliberately does not cancel an external operation.
	return shutdown != "" || w.GetDeletionTimestamp() != nil || phase == "Failed" || phase == "Error", nil
}

// Content-version receipts no longer use a status-sensitive resourceVersion.
// Match the immutable action UID and recorded source digest, never a live object.
// The materialized native Pod manifest is verified above before this check.
// Typed snapshot serialization may add empty native fields, so hashing that body
// is not equivalent to the original source-spec digest.
// Old immutable receipts keep their original UID/RV/digest contract.
func validateFrozenAction(ref api.ResolvedReference, receipt *unstructured.Unstructured) error {
	version, _, _ := unstructured.NestedString(receipt.Object, "spec", "definitionVersions", "action/"+ref.Name)
	if ref.Kind != "ActionDefinition" || ref.Name == "" || ref.UID == "" {
		return errors.New("action does not match frozen submission")
	}
	if strings.HasPrefix(version, ref.UID+":spec:") {
		_, found, err := unstructured.NestedMap(receipt.Object, "spec", "definitions", "actions", ref.Name)
		digest := strings.TrimPrefix(version, ref.UID+":spec:")
		decoded, decodeErr := hex.DecodeString(digest)
		if err != nil || !found || decodeErr != nil || len(decoded) != 32 || (ref.Digest != "" && ref.Digest != digest) {
			return errors.New("action content identity does not match frozen submission")
		}
		return nil
	}
	if ref.ResourceVersion == "" || !strings.HasPrefix(version, ref.UID+":"+ref.ResourceVersion+":") {
		return errors.New("action does not match frozen submission")
	}
	if ref.Digest != "" && version != ref.UID+":"+ref.ResourceVersion+":"+ref.Digest {
		return errors.New("action digest mismatch")
	}
	return nil
}
func (a *adapter) ValidateOutput(_ context.Context, o *api.ResourceOperation, value *api.JSONValue) error {
	if o.Spec.Operation.OutputSchema == nil || value == nil {
		return errors.New("missing output contract or values")
	}
	var s map[string]interface{}
	if json.Unmarshal(o.Spec.Operation.OutputSchema.Raw, &s) != nil || len(p.CheckSchema(s, "")) != 0 {
		return errors.New("invalid output schema")
	}
	loader := gojsonschema.NewSchemaLoader()
	loader.AutoDetect = false
	loader.Draft = gojsonschema.Draft7
	compiled, err := loader.Compile(gojsonschema.NewGoLoader(s))
	if err != nil {
		return errors.New("invalid output schema")
	}
	result, err := compiled.Validate(gojsonschema.NewBytesLoader(value.Raw))
	if err != nil || !result.Valid() {
		return errors.New("output contract rejected")
	}
	return nil
}
func (a *adapter) request(ctx context.Context, method string, s api.OperationSnapshot) (op.Observation, error) {
	var body io.Reader
	if method == http.MethodPost {
		b, _ := json.Marshal(s)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.endpoint+"/operations/"+url.PathEscape(s.IdempotencyKey), body)
	if err != nil {
		return op.Observation{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := a.client.Do(req)
	if err != nil {
		return op.Observation{}, errors.New("provider transport unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound && method == http.MethodGet {
		return op.Observation{}, nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return op.Observation{}, fmt.Errorf("provider HTTP %d", res.StatusCode)
	}
	var out op.Observation
	err = json.NewDecoder(io.LimitReader(res.Body, 65536)).Decode(&out)
	if err != nil {
		return out, errors.New("invalid provider receipt")
	}
	return out, nil
}
func (a *adapter) Lookup(c context.Context, s api.OperationSnapshot) (op.Observation, error) {
	return a.request(c, http.MethodGet, s)
}
func (a *adapter) Start(c context.Context, s api.OperationSnapshot) (op.Observation, error) {
	return a.request(c, http.MethodPost, s)
}
func (a *adapter) Cancel(c context.Context, s api.OperationSnapshot, _ string) error {
	_, e := a.request(c, http.MethodDelete, s)
	return e
}
func (a *adapter) CanCancel() bool { return true }
func main() {
	ns := flag.String("namespace", "", "single managed namespace")
	instance := flag.String("instance", "", "adapter connection identity")
	endpoint := flag.String("endpoint", "", "operator-configured protocol endpoint")
	insecure := flag.Bool("allow-insecure-demo-http", false, "isolated synthetic demo only")
	flag.Parse()
	u, err := url.Parse(*endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (*ns == "" || *instance == "") || !(u.Scheme == "https" || (*insecure && u.Scheme == "http")) {
		log.Fatal("explicit namespace and trusted endpoint required")
	}
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	d, err := dynamic.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	k, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	a := &adapter{instance: *instance, d: d, k: k, ns: *ns, endpoint: strings.TrimRight(*endpoint, "/"), client: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	r := op.Reconciler{Store: a, Provider: a, Verifier: a}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			list, err := d.Resource(operations).Namespace(*ns).List(ctx, metav1.ListOptions{LabelSelector: api.GroupName + "/adapter=" + a.instance})
			if err != nil {
				log.Print("operation list unavailable")
				continue
			}
			for _, item := range list.Items {
				var o api.ResourceOperation
				if runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &o) != nil {
					continue
				}
				if err := r.Reconcile(ctx, &o); err != nil {
					log.Printf("operation %s: %v", o.Name, err)
				}
			}
		}
	}
}
