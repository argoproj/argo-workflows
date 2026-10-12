package parameterization

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

const actionReplacementAnnotation = p.Group + "/action-replacement-policy"

var reviewedImage = regexp.MustCompile(`^[^{}[:space:]]+@sha256:[0-9a-f]{64}$`)

// ActionReplacementPolicy belongs to the original native leaf, not the binding
// or the replacement. This is an author review, not proof of arbitrary code.
type ActionReplacementPolicy struct {
	Contract              parameterapi.ActionExecutionContract `json:"contract"`
	Interface             ActionInterfacePolicy                `json:"interface"`
	ImplementationDigests []string                             `json:"implementationDigests"`
}

type ActionInterfacePolicy struct {
	InputMappings          []parameterapi.ActionParameterMapping `json:"inputMappings,omitempty"`
	OutputMappings         []parameterapi.ActionParameterMapping `json:"outputMappings,omitempty"`
	InputArtifactMappings  []parameterapi.ActionParameterMapping `json:"inputArtifactMappings,omitempty"`
	OutputArtifactMappings []parameterapi.ActionParameterMapping `json:"outputArtifactMappings,omitempty"`
}

type ActionReview struct {
	ImplementationDigest string            `json:"implementationDigest"`
	Definition           ActionDefinition  `json:"definition"`
	Versions             map[string]string `json:"versions"`
	Verification         string            `json:"verification"`
}

func validateActionExecutionContract(c *parameterapi.ActionExecutionContract, t *wfv1.Template) error {
	if c == nil {
		return nil
	}
	if !slices.Contains([]string{"Pure", "ReadOnly", "Mutating"}, c.Effect) {
		return fmt.Errorf("action execution contract requires a recognized effect")
	}
	for _, value := range []string{c.TargetScope, c.PermissionScope, c.OutputMeaning} {
		if strings.TrimSpace(value) == "" || len(value) > 1024 || strings.Contains(value, "{{") {
			return fmt.Errorf("action execution contract identities must be bounded, nonempty and static")
		}
	}
	seen := map[string]bool{}
	for _, capability := range c.Capabilities {
		if seen[capability.Name] || !slices.Contains([]string{"Idempotency", "Observe", "Cancel", "Recover", "ResultReuse"}, capability.Name) || strings.TrimSpace(capability.Condition) == "" || len(capability.Condition) > 2048 {
			return fmt.Errorf("action capability requires a unique recognized name and a bounded applicability condition")
		}
		seen[capability.Name] = true
	}
	images := []string{}
	if t.Container != nil {
		images = append(images, t.Container.Image)
	}
	if t.Script != nil {
		images = append(images, t.Script.Image)
	}
	for _, container := range t.InitContainers {
		images = append(images, container.Image)
	}
	for _, container := range t.Sidecars {
		images = append(images, container.Image)
	}
	for _, image := range images {
		if !reviewedImage.MatchString(image) {
			return fmt.Errorf("reviewed action images must use immutable sha256 references")
		}
	}
	// A native request is never Pure. ReadOnly rejects known mutating verbs; a
	// GET service's behavior still requires author review, not an HTTP inference.
	if t.Resource != nil {
		if c.Effect == "Pure" || c.Effect == "ReadOnly" && t.Resource.Action != "get" {
			return fmt.Errorf("action effect contradicts the native Kubernetes resource verb")
		}
	}
	if t.HTTP != nil {
		method := t.HTTP.Method
		if method == "" {
			method = "GET"
		}
		if c.Effect == "Pure" || c.Effect == "ReadOnly" && method != "GET" && method != "HEAD" && method != "OPTIONS" {
			return fmt.Errorf("action effect contradicts the native HTTP method")
		}
	}
	return nil
}

func actionImplementationDigest(def ActionDefinition) string {
	// The local template name is assigned by the compiler. Everything else,
	// including schemas, declared scopes, capabilities and native security
	// settings, belongs to the exact implementation being reviewed.
	def.Template = def.Template.DeepCopy()
	def.Template.Name = ""
	def.ImplementationRef = nil
	return p.Digest(def)
}

func checkActionReplacement(baseline *wfv1.Template, def ActionDefinition, call ActionCall) error {
	raw, declared := baseline.Metadata.Annotations[actionReplacementAnnotation]
	if !declared && def.ExecutionContract == nil {
		return nil
	} // Legacy, explicitly unverified.
	if !declared || def.ExecutionContract == nil {
		return fmt.Errorf("action replacement requires both an execution contract and an author-approved baseline policy")
	}
	var policy ActionReplacementPolicy
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		return fmt.Errorf("invalid action replacement policy")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("invalid action replacement policy")
	}
	if err := validateActionExecutionContract(&policy.Contract, baseline); err != nil {
		return err
	}
	if err := validateActionExecutionContract(def.ExecutionContract, def.Template); err != nil {
		return err
	}
	want, have := policy.Contract, *def.ExecutionContract
	if want.Effect != have.Effect || want.TargetScope != have.TargetScope || want.PermissionScope != have.PermissionScope || want.OutputMeaning != have.OutputMeaning {
		return fmt.Errorf("action replacement changes effect, target, permission scope or output meaning")
	}
	for _, required := range want.Capabilities {
		if !slices.Contains(have.Capabilities, required) {
			return fmt.Errorf("action replacement does not preserve required capability conditions")
		}
	}
	actualInterface := ActionInterfacePolicy{call.InputMappings, call.OutputMappings, call.InputArtifactMappings, call.OutputArtifactMappings}
	if p.Digest(policy.Interface) != p.Digest(actualInterface) {
		return fmt.Errorf("action replacement interface mappings have not been reviewed by the workflow author")
	}
	digest := actionImplementationDigest(def)
	if len(policy.ImplementationDigests) == 0 || !slices.Contains(policy.ImplementationDigests, digest) {
		return fmt.Errorf("action replacement implementation has not been reviewed by the workflow author")
	}
	return nil
}

// actionReview resolves using the caller's normal Kubernetes permissions. It
// returns review material; it does not approve, grant access or run anything.
func (s Store) actionReview(ctx context.Context, name string) (*ActionReview, error) {
	o, err := s.get(ctx, "actiondefinitions", name)
	if err != nil {
		return nil, err
	}
	var def ActionDefinition
	if err = decode(o, &def); err != nil {
		return nil, err
	}
	versions := map[string]string{"action/" + name: definitionContentVersion(o)}
	closure := &templateClosure{namespace: s.Namespace, versions: versions, sources: map[string]wfv1.TemplateHolder{}}
	def, err = resolveAction(def, func(ref *wfv1.TemplateRef) (*wfv1.Template, error) {
		key, e := closure.source(ctx, ref)
		if e != nil {
			return nil, e
		}
		return actionTemplate(closure.sources[key], ref)
	})
	if err != nil {
		return nil, err
	}
	return &ActionReview{ImplementationDigest: actionImplementationDigest(def), Definition: def, Versions: versions, Verification: "native-contradictions-checked; workflow-author-review-required"}, nil
}
