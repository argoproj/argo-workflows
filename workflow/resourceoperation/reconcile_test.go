package resourceoperation

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
)

type store struct {
	current  *api.ResourceOperation
	conflict bool
}

func (s *store) SaveStatus(_ context.Context, o *api.ResourceOperation) error {
	if s.conflict || s.current.ResourceVersion != o.ResourceVersion {
		return errors.New("conflict")
	}
	v, _ := strconv.Atoi(o.ResourceVersion)
	o.ResourceVersion = strconv.Itoa(v + 1)
	s.current = o.DeepCopy()
	return nil
}

type provider struct {
	obs                            Observation
	lookupErr, startErr, cancelErr error
	starts, cancels                int
	supportsCancel                 bool
}

func (p *provider) Lookup(context.Context, api.OperationSnapshot) (Observation, error) {
	return p.obs, p.lookupErr
}
func (p *provider) Start(context.Context, api.OperationSnapshot) (Observation, error) {
	p.starts++
	p.obs = Observation{Found: true, Phase: "Running", ExternalID: "job-1"}
	return p.obs, p.startErr
}
func (p *provider) Cancel(context.Context, api.OperationSnapshot, string) error {
	p.cancels++
	return p.cancelErr
}
func (p *provider) CanCancel() bool { return p.supportsCancel }

type verifier struct {
	stop                 bool
	verifyErr, outputErr error
}

func (v *verifier) Verify(context.Context, *api.ResourceOperation) (bool, error) {
	return v.stop, v.verifyErr
}
func (v *verifier) ValidateOutput(context.Context, *api.ResourceOperation, *api.JSONValue) error {
	return v.outputErr
}
func fixture() (*Reconciler, *store, *provider, *verifier) {
	s := &store{current: &api.ResourceOperation{ObjectMeta: metav1.ObjectMeta{Name: "operation", ResourceVersion: "1", Generation: 1}, Spec: api.ResourceOperationSpec{Control: "Execute", Operation: api.OperationSnapshot{WorkflowUID: "workflow-uid", NodeID: "invocation-1", IdempotencyKey: "key-1", TerminationPolicy: "Detach"}}}}
	p := &provider{supportsCancel: true}
	v := &verifier{}
	return &Reconciler{Store: s, Provider: p, Verifier: v, Now: func() time.Time { return time.Unix(100, 0) }}, s, p, v
}
func TestLostResponseRecoversWithoutResubmission(t *testing.T) {
	r, s, p, _ := fixture()
	p.startErr = errors.New("lost response")
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	require.Equal(t, "Unknown", s.current.Status.Phase)
	require.Equal(t, 1, p.starts)
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	require.Equal(t, "Running", s.current.Status.Phase)
	require.Equal(t, 1, p.starts)
	p.obs.Phase = "Succeeded"
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	require.Equal(t, "Succeeded", s.current.Status.Phase)
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	require.Equal(t, 1, p.starts)
}
func TestClaimCrashAndMissingReceiptNeverResubmits(t *testing.T) {
	r, s, p, _ := fixture()
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	p.obs = Observation{}
	for range 3 {
		require.NoError(t, r.Reconcile(context.Background(), s.current))
	}
	require.Equal(t, 1, p.starts)
	require.Equal(t, "Unknown", s.current.Status.Phase)
	require.True(t, meta.IsStatusConditionTrue(s.current.Status.Conditions, "OutcomeUnknown"))
}
func TestCASConflictCannotDispatch(t *testing.T) {
	r, s, p, _ := fixture()
	s.conflict = true
	require.Error(t, r.Reconcile(context.Background(), s.current))
	require.Zero(t, p.starts)
}
func TestCancelIsRequestedOnceAndExternallyConfirmed(t *testing.T) {
	r, s, p, _ := fixture()
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	s.current.Spec.Control = "Cancel"
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	require.Equal(t, 1, p.cancels)
	require.Equal(t, "Running", s.current.Status.Phase)
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	require.Equal(t, 1, p.cancels)
	p.obs.Phase = "Cancelled"
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	require.Equal(t, "Cancelled", s.current.Status.Phase)
}
func TestCancelLostResponseIsNotReportedAsCancelled(t *testing.T) {
	r, s, p, _ := fixture()
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	s.current.Spec.Control = "Cancel"
	p.cancelErr = errors.New("lost")
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	require.Equal(t, "Unknown", s.current.Status.Phase)
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	require.Equal(t, 1, p.cancels)
	require.Equal(t, "Running", s.current.Status.Phase)
}
func TestCancellationAndDetach(t *testing.T) {
	for _, policy := range []string{"Detach", "CancelIfSupported"} {
		t.Run(policy, func(t *testing.T) {
			r, s, p, v := fixture()
			s.current.Spec.Operation.TerminationPolicy = policy
			require.NoError(t, r.Reconcile(context.Background(), s.current))
			v.stop = true
			require.NoError(t, r.Reconcile(context.Background(), s.current))
			if policy == "Detach" {
				require.Zero(t, p.cancels)
				require.True(t, meta.IsStatusConditionTrue(s.current.Status.Conditions, "Detached"))
			} else {
				require.Equal(t, 1, p.cancels)
			}
		})
	}
	r, s, p, v := fixture()
	v.stop = true
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	require.Zero(t, p.starts)
	require.Equal(t, "Cancelled", s.current.Status.Phase)
}
func TestObserveAndFailedLookupCannotDispatch(t *testing.T) {
	r, s, p, _ := fixture()
	s.current.Spec.Control = "Observe"
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	require.Zero(t, p.starts)
	s.current.Spec.Control = "Execute"
	p.lookupErr = errors.New("unavailable")
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	require.Zero(t, p.starts)
}
func TestValidationDriftAndOutputContract(t *testing.T) {
	r, s, p, v := fixture()
	v.verifyErr = errors.New("forged invocation")
	require.Error(t, r.Reconcile(context.Background(), s.current))
	require.Zero(t, p.starts)
	v.verifyErr = nil
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	s.current.Spec.Operation.IdempotencyKey = "changed"
	require.Error(t, r.Reconcile(context.Background(), s.current))
	require.Equal(t, 1, p.starts)
	s.current.Spec.Operation.IdempotencyKey = "key-1"
	p.obs.Phase = "Succeeded"
	v.outputErr = errors.New("invalid")
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	require.Equal(t, "Failed", s.current.Status.Phase)
	require.Nil(t, s.current.Status.Outputs)
}
func TestExternalIdentityCannotChange(t *testing.T) {
	r, s, p, _ := fixture()
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	p.obs.ExternalID = "other"
	require.NoError(t, r.Reconcile(context.Background(), s.current))
	require.Equal(t, "Unknown", s.current.Status.Phase)
	require.Equal(t, "job-1", s.current.Status.ExternalOperationID)
}

func TestSnapshotDigestPreservesJSONSemantics(t *testing.T) {
	_, s, _, _ := fixture()
	left := s.current.Spec.Operation
	right := left
	left.Inputs.Raw = []byte(`{"n":12345678901234567890,"a":false}`)
	right.Inputs.Raw = []byte(`{"a":false,"n":12345678901234567890}`)
	a, err := digest(left)
	require.NoError(t, err)
	b, err := digest(right)
	require.NoError(t, err)
	require.Equal(t, a, b)
	right.Inputs.Raw = []byte(`{"a":false,"n":12345678901234567891}`)
	b, err = digest(right)
	require.NoError(t, err)
	require.NotEqual(t, a, b)
	right.Inputs.Raw = []byte(`bad`)
	_, err = digest(right)
	require.Error(t, err)
}
