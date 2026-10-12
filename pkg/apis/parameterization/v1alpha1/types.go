package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// ParameterTemplateSpec describes a business-independent parameter model.
// Only the schema documents are open JSON; the surrounding contract is typed.
type ParameterTemplateSpec struct {
	// Dialect identifies the validation semantics. Unsupported dialects are rejected.
	// +kubebuilder:validation:Enum="http://json-schema.org/draft-07/schema#"
	Dialect string `json:"dialect"`
	// Schema is a JSON Schema document, interpreted by the common resolver.
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	Schema         JSONValue      `json:"schema"`
	IdentityRules  []IdentityRule `json:"identityRules,omitempty"`
	Derived        []Derivation   `json:"derived,omitempty"`
	SensitivePaths []string       `json:"sensitivePaths,omitempty"`
	// SecretSchemas validate plaintext only within the authorized secret backend.
	// +kubebuilder:pruning:PreserveUnknownFields
	SecretSchemas map[string]JSONValue `json:"secretSchemas,omitempty"`
}

type IdentityRule struct {
	Path    string `json:"path"`
	KeyPath string `json:"keyPath"`
}

type Derivation struct {
	Path string `json:"path"`
	From string `json:"from"`
}

// ParameterSetSpec contains a mutable draft and its committed revision pointer.
type ParameterSetSpec struct {
	// +kubebuilder:validation:MinLength=1
	Template string `json:"template"`
	// Values is the only editable business input document. It is validated against Template.
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	Values   JSONValue `json:"values"`
	Revision string    `json:"revision,omitempty"`
	Digest   string    `json:"digest,omitempty"`
}

// ParameterRevisionSpec is an immutable input/model snapshot; it cannot execute a Workflow.
type ParameterRevisionSpec struct {
	Set      string                `json:"set"`
	Parent   string                `json:"parent,omitempty"`
	Template string                `json:"template"`
	Model    ParameterTemplateSpec `json:"model"`
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	Values  JSONValue         `json:"values"`
	Sources map[string]string `json:"sources"`
	// +nullable
	Errors       []FieldError `json:"errors"`
	Digest       string       `json:"digest"`
	Actor        string       `json:"actor"`
	SavedAt      metav1.Time  `json:"savedAt"`
	Message      string       `json:"message,omitempty"`
	RestoredFrom string       `json:"restoredFrom,omitempty"`
}

type FieldError struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// WorkflowFormSpec describes presentation only; values remain in ParameterSet.
type WorkflowFormSpec struct {
	ParameterTemplate string                  `json:"parameterTemplate,omitempty"`
	Labels            map[string]string       `json:"labels,omitempty"`
	Options           map[string]OptionSource `json:"options,omitempty"`
	// Fields and flat parent-linked layout nodes allow arbitrary nesting without recursive API types.
	Fields []FormField      `json:"fields,omitempty"`
	Layout []FormLayoutNode `json:"layout,omitempty"`
}

type OptionSource struct {
	// +kubebuilder:validation:Enum=configmap
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	Key         string   `json:"key"`
	AllowManual bool     `json:"allowManual,omitempty"`
	DependsOn   []string `json:"dependsOn,omitempty"`
}

type FormField struct {
	Path        string `json:"path"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	// +kubebuilder:validation:Enum=text;number;checkbox;select;secret;object;array;table;yaml
	Widget       string          `json:"widget,omitempty"`
	ReadOnly     bool            `json:"readOnly,omitempty"`
	Group        string          `json:"group,omitempty"`
	OptionSource string          `json:"optionSource,omitempty"`
	VisibleWhen  *FieldPredicate `json:"visibleWhen,omitempty"`
	EnabledWhen  *FieldPredicate `json:"enabledWhen,omitempty"`
}

type FieldPredicate struct {
	Path string `json:"path"`
	// +kubebuilder:validation:Enum=Equals;Exists
	Operator string `json:"operator"`
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	Value *JSONValue `json:"value,omitempty"`
}

type FormLayoutNode struct {
	ID     string `json:"id"`
	Parent string `json:"parent,omitempty"`
	// +kubebuilder:validation:Enum=group;tab;tree
	Kind      string `json:"kind"`
	Label     string `json:"label,omitempty"`
	Collapsed bool   `json:"collapsed,omitempty"`
}

// WorkflowParameterBindingSpec maps model paths to native Workflow parameters.
type WorkflowParameterBindingSpec struct {
	// +kubebuilder:validation:MinLength=1
	WorkflowTemplate string `json:"workflowTemplate"`
	// ClusterScope selects a native ClusterWorkflowTemplate; the binding and
	// parameter versions remain namespaced. Defaults to WorkflowTemplate.
	ClusterScope bool `json:"clusterScope,omitempty"`
	// +kubebuilder:validation:MinLength=1
	ParameterTemplate string             `json:"parameterTemplate"`
	ActionBinding     string             `json:"actionBinding,omitempty"`
	Form              string             `json:"form,omitempty"`
	Secrets           []SecretBinding    `json:"secrets,omitempty"`
	Mappings          []ParameterMapping `json:"mappings"`
	Inputs            []NodeInputBinding `json:"inputs,omitempty"`
}

type ParameterMapping struct {
	Path string `json:"path"`
	// +kubebuilder:validation:MinLength=1
	Parameter string `json:"parameter"`
	// +kubebuilder:validation:Enum=json;string;boolean;decimal;secret-reference-json
	Encoding string `json:"encoding"`
}

type SecretBinding struct {
	Path        string `json:"path"`
	Template    string `json:"template"`
	Environment string `json:"environment"`
}

type NodeInputBinding struct {
	Template          string             `json:"template"`
	ParameterTemplate string             `json:"parameterTemplate"`
	Form              string             `json:"form,omitempty"`
	Mappings          []ParameterMapping `json:"mappings"`
}

type FrozenInputForm struct {
	Model    ParameterTemplateSpec `json:"model"`
	Form     *WorkflowFormSpec     `json:"form,omitempty"`
	Mappings []ParameterMapping    `json:"mappings"`
}

// ActionDefinitionSpec freezes an input/output contract and a native Argo implementation.
// +kubebuilder:validation:XValidation:rule="has(self.template) != has(self.implementationRef)",message="exactly one of template or implementationRef is required"
type ActionDefinitionSpec struct {
	// Results binds public execution outputs to reusable result views. External
	// protocols remain in the referenced native implementation or its operator.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	Results []ActionResultDefinition `json:"results,omitempty"`
	// ExecutionContract declares reviewed behavior, not additional RBAC permissions.
	// A replacement with this contract requires the source template author's approval
	// of its resolved implementation digest. Omitted contracts retain legacy behavior.
	ExecutionContract *ActionExecutionContract `json:"executionContract,omitempty"`
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	InputSchema JSONValue `json:"inputSchema"`
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	OutputSchema JSONValue `json:"outputSchema"`
	// Template retains native Argo template semantics and validation.
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	Template *wfv1.Template `json:"template,omitempty"`
	// ImplementationRef selects a native leaf template with the caller's permissions.
	// Exactly one of template or implementationRef is required.
	ImplementationRef *wfv1.TemplateRef `json:"implementationRef,omitempty"`
}

// ActionExecutionContract is an author's semantic assertion about one implementation.
// The server checks native contradictions and exact reviewed implementation identity;
// it cannot infer arbitrary container or remote service behavior from source metadata.
type ActionExecutionContract struct {
	// +kubebuilder:validation:Enum=Pure;ReadOnly;Mutating
	Effect string `json:"effect"`
	// TargetScope and PermissionScope are stable contract identities, not credentials
	// or grants. Their enforcement remains with the native executor and target system.
	// +kubebuilder:validation:MinLength=1
	TargetScope string `json:"targetScope"`
	// +kubebuilder:validation:MinLength=1
	PermissionScope string `json:"permissionScope"`
	// OutputMeaning identifies the semantic meaning of the public output interface.
	// +kubebuilder:validation:MinLength=1
	OutputMeaning string `json:"outputMeaning"`
	// +listType=map
	// +listMapKey=name
	Capabilities []ActionCapability `json:"capabilities,omitempty"`
}

type ActionCapability struct {
	// +kubebuilder:validation:Enum=Idempotency;Observe;Cancel;Recover;ResultReuse
	Name string `json:"name"`
	// Condition states the applicability reviewed by the workflow author.
	// It does not automatically enable retry, cancellation or result reuse.
	// +kubebuilder:validation:MinLength=1
	Condition string `json:"condition"`
}

type WorkflowActionBindingSpec struct {
	WorkflowTemplate string       `json:"workflowTemplate"`
	ClusterScope     bool         `json:"clusterScope,omitempty"`
	Calls            []ActionCall `json:"calls"`
}

// ActionCall selects one author-approved native call site, or a legacy template replacement.
// +kubebuilder:validation:XValidation:rule="has(self.template) != has(self.callPath)",message="exactly one of template or callPath is required"
type ActionCall struct {
	// Template is the legacy whole-template binding. Prefer CallPath for new bindings.
	Template string `json:"template,omitempty"`
	// CallPath starts at the entrypoint template, followed by named DAG tasks or steps.
	// It selects a call site, not one iteration or an observed node ID.
	// +kubebuilder:validation:MinItems=2
	// +kubebuilder:validation:MaxItems=32
	CallPath         []string `json:"callPath,omitempty"`
	ActionDefinition string   `json:"actionDefinition"`
	// InputMappings maps baseline input parameter names to action input names.
	InputMappings []ActionParameterMapping `json:"inputMappings,omitempty"`
	// OutputMappings maps action output names to baseline output parameter names.
	OutputMappings []ActionParameterMapping `json:"outputMappings,omitempty"`
	// Artifact mappings preserve native artifact references; file contents never become parameters.
	InputArtifactMappings  []ActionParameterMapping `json:"inputArtifactMappings,omitempty"`
	OutputArtifactMappings []ActionParameterMapping `json:"outputArtifactMappings,omitempty"`
	// ResultBindings, when present, replaces default action results for this call.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	ResultBindings []ActionResultBinding `json:"resultBindings,omitempty"`
}

// ActionParameterMapping preserves native parameter wire values. Action schemas
// declare string or JSON decoding; this mapping does not evaluate expressions.
type ActionParameterMapping struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ExecutionDefinitions is an immutable snapshot of the definitions used by a
// submission or checked retry. Resource versions identify these snapshots but
// are not a substitute for their historical content.
type ExecutionDefinitions struct {
	ParameterBinding WorkflowParameterBindingSpec    `json:"parameterBinding"`
	ActionBinding    *WorkflowActionBindingSpec      `json:"actionBinding,omitempty"`
	Actions          map[string]ActionDefinitionSpec `json:"actions,omitempty"`
	// CallResults contains resolved contracts keyed by the selected call path.
	CallResults     map[string][]ActionResultDefinition `json:"callResults,omitempty"`
	ResultTemplates map[string]ActionResultTemplateSpec `json:"resultTemplates,omitempty"`
	Form            *WorkflowFormSpec                   `json:"form,omitempty"`
	Versions        map[string]string                   `json:"versions"`
}

// WorkflowSubmissionSpec is an idempotency receipt, never another execution engine.
type WorkflowSubmissionSpec struct {
	RequestDigest   string `json:"requestDigest"`
	Revision        string `json:"revision"`
	Binding         string `json:"binding"`
	ExecutionDigest string `json:"executionDigest"`
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:EmbeddedResource
	Workflow           wfv1.Workflow              `json:"workflow"`
	DefinitionVersions map[string]string          `json:"definitionVersions"`
	InputForms         map[string]FrozenInputForm `json:"inputForms,omitempty"`
	SourceExecution    *SourceExecution           `json:"sourceExecution,omitempty"`
	Definitions        *ExecutionDefinitions      `json:"definitions,omitempty"`
}

type SourceExecution struct {
	Workflow string `json:"workflow"`
	UID      string `json:"uid"`
}

type PartialRetryPlan struct {
	// WorkflowSpecDigest identifies the adopted native execution contract, excluding suspend/shutdown controls.
	WorkflowSpecDigest string `json:"workflowSpecDigest,omitempty"`
	// CallPaths optionally limits an explicit retry to declared invocation paths and their downstream closure.
	CallPaths         []string          `json:"callPaths,omitempty"`
	ExecutionDigest   string            `json:"executionDigest,omitempty"`
	Workflow          string            `json:"workflow"`
	UID               string            `json:"uid"`
	ResourceVersion   string            `json:"resourceVersion"`
	Revision          string            `json:"revision"`
	PreviousRevision  string            `json:"previousRevision"`
	Binding           string            `json:"binding"`
	ChangedParameters []string          `json:"changedParameters"`
	AffectedTasks     []string          `json:"affectedTasks"`
	RetainedNodes     map[string]string `json:"retainedNodes"`
	Selector          string            `json:"selector"`
	Parameters        []string          `json:"parameters"`
	Digest            string            `json:"digest"`
}

type ParameterRetrySpec struct {
	Definitions      *ExecutionDefinitions `json:"definitions,omitempty"`
	PreviousRevision string                `json:"previousRevision,omitempty"`
	Plan             *PartialRetryPlan     `json:"plan,omitempty"`
	// PreviousWorkflow preserves inputs and outputs removed by native retry.
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:EmbeddedResource
	PreviousWorkflow *wfv1.Workflow `json:"previousWorkflow,omitempty"`

	RequestDigest   string `json:"requestDigest"`
	Workflow        string `json:"workflow"`
	WorkflowUID     string `json:"workflowUID"`
	ResourceVersion string `json:"resourceVersion"`
	Revision        string `json:"revision"`
	Actor           string `json:"actor"`
}

type ParameterNodeInputSpec struct {
	// OptionVersions records the allowed choices' source and dependency versions at input commit.
	OptionVersions map[string]string `json:"optionVersions,omitempty"`
	// SecretVersions pins immutable references; it never contains secret values.
	SecretVersions map[string]string `json:"secretVersions,omitempty"`
	RequestDigest  string            `json:"requestDigest"`
	WorkflowUID    string            `json:"workflowUID"`
	Workflow       string            `json:"workflow"`
	NodeID         string            `json:"nodeId"`
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	Values      JSONValue         `json:"values"`
	Sources     map[string]string `json:"sources"`
	Contract    FrozenInputForm   `json:"contract"`
	Actor       string            `json:"actor"`
	SubmittedAt *metav1.Time      `json:"submittedAt,omitempty"`
}

// DefinitionStatus reports validation of one observed spec generation.
// It never grants permission to submit or execute an action.
type DefinitionStatus struct {
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// +listType=map
	// +listMapKey=type
	Conditions         []metav1.Condition  `json:"conditions,omitempty"`
	ResolvedReferences []ResolvedReference `json:"resolvedReferences,omitempty"`
	Errors             []FieldError        `json:"errors,omitempty"`
	Digest             string              `json:"digest,omitempty"`
}

type ResolvedReference struct {
	Kind            string `json:"kind"`
	Name            string `json:"name"`
	UID             string `json:"uid"`
	ResourceVersion string `json:"resourceVersion"`
	Digest          string `json:"digest,omitempty"`
}

// ResourceOperationSpec is the optional contract for one external operation.
// Its controller is separate from Workflow scheduling; an absent controller leaves it unobserved.
type ResourceOperationSpec struct {
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="operation snapshot is immutable"
	Operation OperationSnapshot `json:"operation"`
	// +kubebuilder:validation:Enum=Observe;Execute;Cancel
	// +kubebuilder:default=Observe
	Control string `json:"control,omitempty"`
}

type OperationSnapshot struct {
	// OutputSchema is the frozen provider result contract, not a mutable definition lookup.
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	OutputSchema *JSONValue `json:"outputSchema,omitempty"`
	// TerminationPolicy applies when the owning Workflow stops, expires or disappears.
	// +kubebuilder:validation:Enum=Detach;CancelIfSupported
	// +kubebuilder:default=Detach
	TerminationPolicy string            `json:"terminationPolicy,omitempty"`
	ActionRef         ResolvedReference `json:"actionRef"`
	Target            OperationTarget   `json:"target"`
	IdempotencyKey    string            `json:"idempotencyKey"`
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	Inputs      JSONValue `json:"inputs"`
	WorkflowUID string    `json:"workflowUID"`
	NodeID      string    `json:"nodeId"`
	Attempt     int32     `json:"attempt"`
}

type OperationTarget struct {
	ConnectionRef string `json:"connectionRef"`
	ResourceType  string `json:"resourceType"`
	ResourceID    string `json:"resourceId,omitempty"`
}

type ResourceOperationStatus struct {
	// RequestDigest detects snapshot drift after a durable dispatch claim.
	RequestDigest      string `json:"requestDigest,omitempty"`
	ObservedGeneration int64  `json:"observedGeneration,omitempty"`
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed;Unknown;Cancelled
	Phase               string `json:"phase,omitempty"`
	ExternalOperationID string `json:"externalOperationId,omitempty"`
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	Outputs *JSONValue `json:"outputs,omitempty"`
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	StartedAt  *metav1.Time       `json:"startedAt,omitempty"`
	FinishedAt *metav1.Time       `json:"finishedAt,omitempty"`
}

// JSONValue preserves a schema or input document without converting numbers or null.
// +kubebuilder:validation:Schemaless
// +kubebuilder:pruning:PreserveUnknownFields
type JSONValue struct {
	Raw json.RawMessage `json:"-"`
}

// DeepCopyInto keeps the copy expressed in terms of the public RawMessage type.
// Newer Go versions implement RawMessage as an alias; generated code must not
// expose that implementation type to builds using an older supported Go version.
func (v *JSONValue) DeepCopyInto(out *JSONValue) {
	*out = *v
	if v.Raw != nil {
		out.Raw = make(json.RawMessage, len(v.Raw))
		copy(out.Raw, v.Raw)
	}
}

func (v JSONValue) MarshalJSON() ([]byte, error) {
	if len(v.Raw) == 0 {
		return []byte("null"), nil
	}
	return v.Raw.MarshalJSON()
}
func (v *JSONValue) UnmarshalJSON(data []byte) error { return v.Raw.UnmarshalJSON(data) }
