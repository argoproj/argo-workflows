package parameterization

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

// latestSecretRef is a reference-only version index, not a mutable credential.
// Its ConfigMap key contains {"name":"immutable-secret-version","key":"value"}.
// Both reads use caller RBAC in the execution namespace. Preview has no writes.
// Submit re-resolves and checks the digest; native retries retain the pinned
// version. A reviewed new-parameter retry may explicitly resolve the index again.
func (s Store) resolveLatestSecrets(ctx context.Context, model p.Model, values any, versions map[string]string) error {
	indexes := map[string]*corev1.ConfigMap{}
	for _, pattern := range model.SensitivePaths {
		for _, path := range p.ExpandPaths(values, pattern) {
			raw, ok := p.Pointer(values, path)
			if !ok || raw == nil {
				continue
			}
			input, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			latest, ok := input["latestSecretRef"].(map[string]any)
			if !ok {
				continue
			}
			name, _ := latest["configMap"].(string)
			key, _ := latest["key"].(string)
			if len(input) != 1 || len(latest) != 2 || len(validation.IsDNS1123Subdomain(name)) > 0 || key == "" {
				return fmt.Errorf("invalid latest Secret version reference at %s", path)
			}
			index := indexes[name]
			if index == nil {
				var err error
				index, err = auth.GetKubeClient(ctx).CoreV1().ConfigMaps(s.Namespace).Get(ctx, name, metav1.GetOptions{})
				if err != nil {
					return fmt.Errorf("latest Secret version index is unavailable or forbidden at %s", path)
				}
				indexes[name] = index
			}
			data := index.Data[key]
			ref := map[string]string{}
			if len(data) > 4096 || json.Unmarshal([]byte(data), &ref) != nil || len(ref) != 2 || ref["key"] == "" || len(validation.IsDNS1123Subdomain(ref["name"])) > 0 {
				return fmt.Errorf("latest Secret index must identify one namespaced immutable Secret key at %s", path)
			}
			secret, err := auth.GetKubeClient(ctx).CoreV1().Secrets(s.Namespace).Get(ctx, ref["name"], metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("latest Secret version is unavailable or forbidden at %s", path)
			}
			if leaseErr := p.ValidateExternalSecretLease(secret, time.Now()); leaseErr != nil {
				return leaseErr
			}
			if secret.Immutable == nil || !*secret.Immutable {
				return fmt.Errorf("latest Secret index must resolve an immutable version at %s", path)
			}
			plaintext, exists := secret.Data[ref["key"]]
			if !exists {
				return fmt.Errorf("latest Secret key is absent at %s", path)
			}
			if secretLogicalSchema(model, path) != nil {
				if err = validateSecretValue(model, path, string(plaintext)); err != nil {
					return err
				}
			}
			versions["secret/"+secret.Name] = string(secret.UID) + ":" + secret.ResourceVersion
			versions["latestSecret/"+path] = string(index.UID) + ":" + index.ResourceVersion + ":" + name + "/" + key + "->" + secret.Name + "/" + ref["key"]
			delete(input, "latestSecretRef")
			input["secretRef"] = map[string]any{"name": secret.Name, "key": ref["key"]}
		}
	}
	return nil
}
