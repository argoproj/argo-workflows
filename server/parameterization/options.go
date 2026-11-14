package parameterization

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

type Option struct {
	Value       any            `json:"value"`
	Label       string         `json:"label"`
	Description string         `json:"description,omitempty"`
	Disabled    bool           `json:"disabled,omitempty"`
	When        map[string]any `json:"when,omitempty"`
}
type OptionSource = parameterapi.OptionSource
type OptionResult struct {
	Items         []Option `json:"items"`
	Version       string   `json:"version"`
	ContextDigest string   `json:"contextDigest"`
}

// OptionProvider receives caller-scoped context and cannot be chosen by arbitrary URL.
// Additional backends must be explicitly registered by the server operator.
type OptionProvider interface {
	Query(context.Context, string, OptionSource) ([]Option, string, error)
}
type ConfigMapOptions struct{}

func (ConfigMapOptions) Query(ctx context.Context, ns string, source OptionSource) ([]Option, string, error) {
	cm, err := auth.GetKubeClient(ctx).CoreV1().ConfigMaps(ns).Get(ctx, source.Name, metav1.GetOptions{})
	if err != nil {
		return nil, "", err
	}
	value, exists := cm.Data[source.Key]
	if !exists {
		return nil, "", fmt.Errorf("option data key is absent")
	}
	if len(value) > 262144 {
		return nil, "", fmt.Errorf("option data exceeds 256 KiB")
	}
	var entries []Option
	if err = json.Unmarshal([]byte(value), &entries); err != nil {
		return nil, "", fmt.Errorf("invalid option data")
	}
	if len(entries) > 1000 {
		return nil, "", fmt.Errorf("option source exceeds 1000 entries")
	}
	return entries, string(cm.UID) + ":" + cm.ResourceVersion, nil
}

type OptionsRequest struct {
	Input   *InputContext `json:"input,omitempty"`
	Binding string        `json:"binding,omitempty"`
	Path    string        `json:"path"`
	Values  any           `json:"values"`
}

func (h Handler) options(ctx context.Context, s Store, req OptionsRequest) (*OptionResult, error) {
	if req.Input == nil {
		return s.Options(ctx, req, h.OptionProviders)
	}
	if req.Binding != "" {
		return nil, fmt.Errorf("binding and frozen input context are mutually exclusive")
	}
	contract, err := h.inputContract(ctx, s, *req.Input)
	if err != nil {
		return nil, err
	}
	form, err := inputFormDefinition(contract)
	if err != nil {
		return nil, err
	}
	return s.optionsForForm(ctx, form, req.Path, req.Values, h.OptionProviders)
}

func inputFormDefinition(contract *FrozenInputForm) (parameterapi.WorkflowFormSpec, error) {
	var form parameterapi.WorkflowFormSpec
	raw, err := json.Marshal(contract.Form)
	if err == nil {
		err = json.Unmarshal(raw, &form)
	}
	return form, err
}

func (s Store) Options(ctx context.Context, req OptionsRequest, providers map[string]OptionProvider) (*OptionResult, error) {
	if req.Input != nil {
		return nil, fmt.Errorf("frozen input options require workflow authorization")
	}
	b, err := s.get(ctx, "workflowparameterbindings", req.Binding)
	if err != nil {
		return nil, err
	}
	var binding Binding
	if err = decode(b, &binding); err != nil {
		return nil, err
	}
	if binding.Form == "" {
		return nil, fmt.Errorf("binding has no option sources")
	}
	f, err := s.get(ctx, "workflowforms", binding.Form)
	if err != nil {
		return nil, err
	}
	var form parameterapi.WorkflowFormSpec
	if err = decode(f, &form); err != nil {
		return nil, err
	}
	return s.optionsForForm(ctx, form, req.Path, req.Values, providers)
}

func (s Store) optionsForForm(ctx context.Context, form parameterapi.WorkflowFormSpec, path string, values any, providers map[string]OptionProvider) (*OptionResult, error) {
	if providers == nil {
		providers = map[string]OptionProvider{"configmap": ConfigMapOptions{}}
	}
	sources, err := formOptions(form)
	if err != nil {
		return nil, err
	}
	source, exists := selectedOption(sources, path)
	if !exists {
		return nil, fmt.Errorf("field has no declared option source")
	}
	provider, exists := providers[source.Kind]
	if !exists {
		return nil, fmt.Errorf("unsupported option provider")
	}
	dependencies := map[string]any{}
	for _, path := range source.DependsOn {
		value, ok := p.Pointer(values, path)
		if !ok {
			return nil, fmt.Errorf("option dependency %s is absent", path)
		}
		dependencies[path] = value
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	items, version, err := provider.Query(ctx, s.Namespace, source)
	if err != nil {
		return nil, err
	}
	out := &OptionResult{Items: []Option{}, Version: version, ContextDigest: p.Digest(dependencies)}
	for _, item := range items {
		matches := true
		for path, want := range item.When {
			got, ok := dependencies[path]
			if !ok || p.Digest(want) != p.Digest(got) {
				matches = false
				break
			}
		}
		if matches {
			item.When = nil
			out.Items = append(out.Items, item)
		}
	}
	return out, nil
}

// Recheck choices immediately before committing native inputs. The receipt
// retains source/context versions; later status reads do not rerun providers.
func (s Store) validateInputOptions(ctx context.Context, contract *FrozenInputForm, values any) (map[string]string, error) {
	form, err := inputFormDefinition(contract)
	if err != nil {
		return nil, err
	}
	sources, err := formOptions(form)
	if err != nil {
		return nil, err
	}
	versions := map[string]string{}
	for path, source := range expandFormOptions(sources, values) {
		value, exists := p.Pointer(values, path)
		if !exists {
			continue // An absent optional field does not consume an option provider.
		}
		options, err := s.optionsForForm(ctx, form, path, values, s.OptionProviders)
		if err != nil {
			return nil, err
		}
		if !source.AllowManual {
			allowed := false
			for _, item := range options.Items {
				if !item.Disabled && p.Digest(item.Value) == p.Digest(value) {
					allowed = true
					break
				}
			}
			if !allowed {
				return nil, fmt.Errorf("selected option is unavailable at %s; refresh choices before submitting", path)
			}
		}
		versions[path] = options.Version + ":" + options.ContextDigest
	}
	return versions, nil
}
