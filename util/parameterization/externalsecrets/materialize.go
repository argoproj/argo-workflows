package externalsecrets

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

const SourceLabel = p.Group + "/external-source"
const ValidUntil = p.Group + "/external-valid-until"
const VersionAnnotation = p.Group + "/external-version"

// Materializer only owns Secrets and an index under an administrator-approved
// namespace/name. Rotations never overwrite an immutable version or Workflow.
type Materializer struct {
	Client                    kubernetes.Interface
	Provider                  Provider
	Namespace, Name, SourceID string
	Validity                  time.Duration
	Now                       func() time.Time
}

func (m Materializer) Reconcile(ctx context.Context) error {
	if m.Namespace == "" || m.Name == "" || m.SourceID == "" || m.Validity < 15*time.Second || m.Validity > 10*time.Minute {
		return fmt.Errorf("invalid external secret materializer configuration")
	}
	now := time.Now()
	if m.Now != nil {
		now = m.Now()
	}
	secrets := m.Client.CoreV1().Secrets(m.Namespace)
	owned, err := secrets.List(ctx, metav1.ListOptions{LabelSelector: SourceLabel + "=" + m.SourceID})
	if err != nil {
		return fmt.Errorf("external secret inventory unavailable")
	}
	// Expiration also removes the injection target when the provider is offline.
	// A blocked new Pod cannot obtain a stale materialized credential.
	for _, secret := range owned.Items {
		if p.ValidateExternalSecretLease(&secret, now) != nil {
			uid, rv := secret.UID, secret.ResourceVersion
			if err = secrets.Delete(ctx, secret.Name, metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &uid, ResourceVersion: &rv}}); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("expired external secret removal failed")
			}
		}
	}
	meta, err := m.Provider.Metadata(ctx)
	if err != nil {
		return err
	}
	owned, err = secrets.List(ctx, metav1.ListOptions{LabelSelector: SourceLabel + "=" + m.SourceID})
	if err != nil {
		return fmt.Errorf("external secret inventory unavailable")
	}
	// Revoked versions are removed only after authoritative metadata is read.
	// During an outage, the last successful metadata lease expires in admission.
	for _, secret := range owned.Items {
		state, exists := meta.Versions[secret.Annotations[VersionAnnotation]]
		if !exists || !state.Active(now) {
			uid, rv := secret.UID, secret.ResourceVersion
			if err = secrets.Delete(ctx, secret.Name, metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &uid, ResourceVersion: &rv}}); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("revoked external secret removal failed")
			}
			continue
		}
		secret.Annotations[ValidUntil] = leaseDeadline(now, m.Validity, state)
		if _, err = secrets.Update(ctx, &secret, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("external secret verification lease update failed")
		}
	}
	current, stateExists := meta.Versions[strconv.Itoa(meta.Current)]
	if !stateExists || !current.Active(now) {
		return m.writeIndex(ctx, "")
	}
	name := fmt.Sprintf("external-%s-v%d", m.SourceID, meta.Current)
	secret, err := secrets.Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		value, readErr := m.Provider.Read(ctx, meta.Current)
		if readErr != nil {
			return readErr
		}
		// Recheck metadata after reading plaintext so rotation/deletion races do not
		// publish a latest index for a version already revoked before this commit.
		check, checkErr := m.Provider.Metadata(ctx)
		if checkErr != nil {
			return checkErr
		}
		version, exists := check.Versions[strconv.Itoa(meta.Current)]
		if !exists || !version.Active(now) || check.Current != meta.Current {
			return fmt.Errorf("external secret version changed during materialization")
		}
		immutable := true
		secret = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: m.Namespace, Labels: map[string]string{SourceLabel: m.SourceID}, Annotations: map[string]string{VersionAnnotation: strconv.Itoa(meta.Current), ValidUntil: leaseDeadline(now, m.Validity, version)}}, Immutable: &immutable, Type: corev1.SecretTypeOpaque, Data: map[string][]byte{"value": value}}
		secret, err = secrets.Create(ctx, secret, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			secret, err = secrets.Get(ctx, name, metav1.GetOptions{})
			if err == nil && subtle.ConstantTimeCompare(secret.Data["value"], value) != 1 {
				return fmt.Errorf("external secret version identity conflict")
			}
		}
	}
	if err != nil || secret.Immutable == nil || !*secret.Immutable || secret.Labels[SourceLabel] != m.SourceID || secret.Annotations[VersionAnnotation] != strconv.Itoa(meta.Current) {
		return fmt.Errorf("external secret materialization identity conflict")
	}
	return m.writeIndex(ctx, name)
}
func leaseDeadline(now time.Time, validity time.Duration, state Version) string {
	deadline := now.Add(validity)
	if state.DeletionTime != "" {
		if deletion, err := time.Parse(time.RFC3339Nano, state.DeletionTime); err == nil && deletion.Before(deadline) {
			deadline = deletion
		}
	}
	return deadline.UTC().Format(time.RFC3339Nano)
}
func (m Materializer) writeIndex(ctx context.Context, name string) error {
	client := m.Client.CoreV1().ConfigMaps(m.Namespace)
	data := map[string]string{}
	if name != "" {
		raw, _ := json.Marshal(map[string]string{"name": name, "key": "value"})
		data["current"] = string(raw)
	}
	cm, err := client.Get(ctx, m.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.Create(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: m.Name, Namespace: m.Namespace, Labels: map[string]string{SourceLabel: m.SourceID}}, Data: data}, metav1.CreateOptions{})
	} else if err == nil {
		if cm.Labels[SourceLabel] != m.SourceID {
			return fmt.Errorf("external secret index is owned by another source")
		}
		if p.Digest(cm.Data) == p.Digest(data) {
			return nil
		}
		cm.Data = data
		_, err = client.Update(ctx, cm, metav1.UpdateOptions{})
	}
	if err != nil {
		return fmt.Errorf("external secret index update failed")
	}
	return nil
}
