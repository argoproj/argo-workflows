package spec

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Plugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              PluginSpec `json:"spec"`
}

func (p Plugin) Validate() error {
	if err := p.Spec.Sidecar.Validate(); err != nil {
		return fmt.Errorf("sidecar is invalid: %w", err)
	}
	return nil
}

type PluginSpec struct {
	Sidecar Sidecar `json:"sidecar"`
}

type Sidecar struct {
	// AutomountServiceAccount mounts the service account's token. The service account must have the same name as the plugin.
	AutomountServiceAccountToken bool            `json:"automountServiceAccountToken,omitempty"`
	Container                    apiv1.Container `json:"container"`
}

func (s Sidecar) Validate() error {
	c := s.Container
	if len(c.Ports) < 1 {
		return fmt.Errorf("at least one port is mandatory")
	}
	if c.Resources.Requests == nil {
		return fmt.Errorf("resources requests are mandatory")
	}
	if c.Resources.Limits == nil {
		return fmt.Errorf("resources limits are mandatory")
	}
	if c.SecurityContext == nil {
		return fmt.Errorf("security context is mandatory")
	}
	return nil
}
