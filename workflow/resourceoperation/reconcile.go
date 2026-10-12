// Package resourceoperation provides opt-in single-operation recovery. It never
// schedules a Workflow task or writes Workflow status. Provider adapters retain
// responsibility for their actual idempotency and cancellation guarantees.
package resourceoperation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
)

// Store must perform a resourceVersion compare-and-swap for every status write.
// A failed write must never be followed by an external mutation.
type Store interface {
	SaveStatus(context.Context, *api.ResourceOperation) error
}

type Observation struct {
	Found      bool
	Phase      string
	ExternalID string
	Outputs    *api.JSONValue
}

type Provider interface {
	// Lookup must be scoped to the frozen target and logical key. A missing or
	// unavailable result after dispatch is not permission to submit again.
	Lookup(context.Context, api.OperationSnapshot) (Observation, error)
	Start(context.Context, api.OperationSnapshot) (Observation, error)
	Cancel(context.Context, api.OperationSnapshot, string) error
	CanCancel() bool
}

// Verifier checks trusted caller/execution identity, frozen action/connection
// versions and input contracts. StopRequested is independent of suspend.
type Verifier interface {
	Verify(context.Context, *api.ResourceOperation) (stopRequested bool, err error)
	ValidateOutput(context.Context, *api.ResourceOperation, *api.JSONValue) error
}

type Reconciler struct {
	Store    Store
	Provider Provider
	Verifier Verifier
	Now      func() time.Time
}

func digest(op api.OperationSnapshot) (string, error) {
	b, err := json.Marshal(op)
	if err != nil {
		return "", fmt.Errorf("invalid operation snapshot")
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return "", fmt.Errorf("invalid operation snapshot")
	}
	b, err = json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("invalid operation snapshot")
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
func (r *Reconciler) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

func terminal(p string) bool { return p == "Succeeded" || p == "Failed" || p == "Cancelled" }
func (r *Reconciler) condition(o *api.ResourceOperation, typ, reason string, active bool) {
	status := metav1.ConditionFalse
	if active {
		status = metav1.ConditionTrue
	}
	meta.SetStatusCondition(&o.Status.Conditions, metav1.Condition{Type: typ, Status: status, Reason: reason, Message: reason, ObservedGeneration: o.Generation, LastTransitionTime: metav1.NewTime(r.now())})
}
func (r *Reconciler) save(ctx context.Context, o *api.ResourceOperation) error {
	o.Status.ObservedGeneration = o.Generation
	return r.Store.SaveStatus(ctx, o)
}
func (r *Reconciler) unknown(ctx context.Context, o *api.ResourceOperation, reason string) error {
	o.Status.Phase = "Unknown"
	r.condition(o, "OutcomeUnknown", reason, true)
	return r.save(ctx, o)
}

// Reconcile advances one operation. Dispatch is claimed durably before Start.
// A crash after claim but before Start intentionally requires explicit recovery;
// no adapter-independent exactly-once guarantee can resolve that window.
func (r *Reconciler) Reconcile(ctx context.Context, source *api.ResourceOperation) error {
	o := source.DeepCopy()
	if o.Spec.Control != "" && o.Spec.Control != "Observe" && o.Spec.Control != "Execute" && o.Spec.Control != "Cancel" {
		return fmt.Errorf("invalid operation control")
	}
	if o.Spec.Operation.WorkflowUID == "" || o.Spec.Operation.NodeID == "" || o.Spec.Operation.IdempotencyKey == "" {
		return fmt.Errorf("missing operation identity")
	}
	stop, err := r.Verifier.Verify(ctx, o)
	if err != nil {
		return fmt.Errorf("operation verification failed: %w", err)
	}
	d, err := digest(o.Spec.Operation)
	if err != nil {
		return err
	}
	if o.Status.RequestDigest != "" && o.Status.RequestDigest != d {
		return fmt.Errorf("operation snapshot changed after dispatch")
	}
	if terminal(o.Status.Phase) {
		return nil
	}
	cancel := o.Spec.Control == "Cancel" || (stop && o.Spec.Operation.TerminationPolicy == "CancelIfSupported")
	if stop && o.Spec.Operation.TerminationPolicy != "CancelIfSupported" {
		r.condition(o, "Detached", "WorkflowStoppedExternalOperationRetained", true)
	}
	// Do not start after cancellation or termination, even when lookup is empty.
	obs, err := r.Provider.Lookup(ctx, o.Spec.Operation)
	if err != nil {
		return r.unknown(ctx, o, "LookupUnavailable")
	}
	if obs.Found {
		if o.Status.RequestDigest == "" {
			o.Status.RequestDigest = d
		}
		if obs.ExternalID == "" {
			return r.unknown(ctx, o, "MissingExternalIdentity")
		}
		if o.Status.ExternalOperationID != "" && o.Status.ExternalOperationID != obs.ExternalID {
			return r.unknown(ctx, o, "ExternalIdentityChanged")
		}
		o.Status.ExternalOperationID = obs.ExternalID
		if terminal(obs.Phase) {
			return r.observe(ctx, o, obs)
		}
		if cancel {
			if !r.Provider.CanCancel() {
				r.condition(o, "CancelRequested", "CancellationUnsupported", true)
				return r.observe(ctx, o, obs)
			}
			// A cancellation request also has a durable claim; only Lookup confirms it.
			if !meta.IsStatusConditionTrue(o.Status.Conditions, "CancelRequested") {
				r.condition(o, "CancelRequested", "AwaitingExternalConfirmation", true)
				if err = r.save(ctx, o); err != nil {
					return err
				}
				if err = r.Provider.Cancel(ctx, o.Spec.Operation, obs.ExternalID); err != nil {
					return r.unknown(ctx, o, "CancelResponseUnknown")
				}
				return nil
			}
		}
		return r.observe(ctx, o, obs)
	}
	if o.Status.StartedAt != nil {
		return r.unknown(ctx, o, "DispatchedOperationNotFound")
	}
	if cancel || stop {
		o.Status.Phase = "Cancelled"
		t := metav1.NewTime(r.now())
		o.Status.FinishedAt = &t
		r.condition(o, "Cancelled", "CancelledBeforeDispatch", true)
		return r.save(ctx, o)
	}
	if o.Spec.Control != "Execute" {
		o.Status.Phase = "Pending"
		return r.save(ctx, o)
	}
	o.Status.RequestDigest = d
	o.Status.Phase = "Running"
	t := metav1.NewTime(r.now())
	o.Status.StartedAt = &t
	r.condition(o, "DispatchClaimed", "PersistedBeforeExternalRequest", true)
	if err = r.save(ctx, o); err != nil {
		return err
	}
	obs, err = r.Provider.Start(ctx, o.Spec.Operation)
	if err != nil {
		return r.unknown(ctx, o, "StartResponseUnknown")
	}
	if !obs.Found || obs.ExternalID == "" {
		return r.unknown(ctx, o, "StartReceiptMissing")
	}
	return r.observe(ctx, o, obs)
}
func (r *Reconciler) observe(ctx context.Context, o *api.ResourceOperation, obs Observation) error {
	switch obs.Phase {
	case "Pending", "Running", "Succeeded", "Failed", "Cancelled":
	default:
		return r.unknown(ctx, o, "UnrecognizedExternalState")
	}
	if obs.Phase == "Succeeded" {
		if err := r.Verifier.ValidateOutput(ctx, o, obs.Outputs); err != nil {
			o.Status.Phase = "Failed"
			o.Status.Outputs = nil
			t := metav1.NewTime(r.now())
			o.Status.FinishedAt = &t
			r.condition(o, "OutputValid", "OutputContractRejected", false)
			return r.save(ctx, o)
		}
		o.Status.Outputs = obs.Outputs
		r.condition(o, "OutputValid", "OutputContractAccepted", true)
	}
	o.Status.ExternalOperationID = obs.ExternalID
	o.Status.Phase = obs.Phase
	r.condition(o, "OutcomeUnknown", "ObservedExternalState", false)
	if terminal(obs.Phase) {
		t := metav1.NewTime(r.now())
		o.Status.FinishedAt = &t
	}
	if obs.Phase == "Cancelled" {
		r.condition(o, "Cancelled", "ExternallyConfirmed", true)
	}
	return r.save(ctx, o)
}
