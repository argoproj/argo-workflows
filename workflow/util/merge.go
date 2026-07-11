package util

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// allowedUserOverrideFields lists WorkflowSpec fields that users may set when
// submitting a workflow via workflowTemplateRef under Strict/Secure mode.
// Any field NOT in this set is blocked by default, ensuring new fields added
// to WorkflowSpec are denied until explicitly reviewed.
var allowedUserOverrideFields = map[string]bool{
	"Arguments":             true,
	"Entrypoint":            true,
	"Shutdown":              true,
	"Suspend":               true,
	"ActiveDeadlineSeconds": true,
	"Priority":              true,
	"TTLStrategy":           true,
	"PodGC":                 true,
	"VolumeClaimGC":         true,
	"ArchiveLogs":           true,
	"WorkflowMetadata":      true,
	"WorkflowTemplateRef":   true,
	"Metrics":               true,
	"ArtifactGC":            true,
}

// userOverrideAllowlistEnv names an env var operators may set to ADD WorkflowSpec
// field names to allowedUserOverrideFields (comma-separated, e.g.
// "podSpecPatch,volumes"). Field names are the YAML/JSON names operators write in
// their workflows, not Go identifiers. This re-opens overrides that are blocked by
// default, so it is opt-in and validated against real field names.
// Note: the nested ArtifactGC.PodSpecPatch/ServiceAccountName/PodMetadata blocks
// are enforced separately and are NOT relaxed by this var.
const userOverrideAllowlistEnv = "WORKFLOW_USER_OVERRIDE_ALLOWLIST"

// ConfigureUserOverrideAllowlistFromEnv adds the WorkflowSpec field names listed
// in $WORKFLOW_USER_OVERRIDE_ALLOWLIST to allowedUserOverrideFields. It is
// controller-only configuration and must be called once at controller startup,
// so an invalid field name fails the controller rather than any binary (CLI,
// argoexec) that merely imports this package.
func ConfigureUserOverrideAllowlistFromEnv() error {
	fields, err := parseUserOverrideAllowlist(os.Getenv(userOverrideAllowlistEnv))
	if err != nil {
		return err
	}
	for _, f := range fields {
		allowedUserOverrideFields[f] = true
	}
	return nil
}

// parseUserOverrideAllowlist parses a comma-separated list of WorkflowSpec field
// names (the YAML/JSON names operators write, e.g. "podSpecPatch"), returning the
// corresponding Go field names that key allowedUserOverrideFields. It errors on any
// name that is not a real WorkflowSpec field so an operator typo is surfaced rather
// than silently leaving a field blocked.
func parseUserOverrideAllowlist(env string) ([]string, error) {
	if strings.TrimSpace(env) == "" {
		return nil, nil
	}
	// Map YAML/JSON name -> Go field name; allowedUserOverrideFields is keyed by Go name.
	goName := map[string]string{}
	t := reflect.TypeFor[wfv1.WorkflowSpec]()
	for field := range t.Fields() {
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			name = field.Name // ponytail: fall back to Go name for any untagged field
		}
		goName[name] = field.Name
	}
	var fields []string
	for f := range strings.SplitSeq(env, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		g, ok := goName[f]
		if !ok {
			return nil, fmt.Errorf("%s: %q is not a WorkflowSpec field name", userOverrideAllowlistEnv, f)
		}
		fields = append(fields, g)
	}
	return fields, nil
}

// ValidateUserOverrides checks that a user-submitted WorkflowSpec only sets
// fields from the allow-list. Returns an error listing all violations.
func ValidateUserOverrides(userSpec *wfv1.WorkflowSpec) error {
	if userSpec == nil {
		return nil
	}
	v := reflect.ValueOf(userSpec).Elem()
	t := v.Type()
	zero := reflect.New(t).Elem()

	var violations []string
	for i := 0; i < t.NumField(); i++ {
		fieldName := t.Field(i).Name
		if allowedUserOverrideFields[fieldName] {
			continue
		}
		if !reflect.DeepEqual(v.Field(i).Interface(), zero.Field(i).Interface()) {
			violations = append(violations, fieldName)
		}
	}
	// ArtifactGC is allow-listed so that its benign fields (Strategy,
	// ForceFinalizerRemoval) may be set, but its nested ServiceAccountName,
	// PodSpecPatch and PodMetadata reach the artifact-GC Pod and would otherwise
	// re-open the privilege escalation that the top-level ServiceAccountName /
	// PodSpecPatch / PodMetadata blocks are meant to close, so reject them here.
	violations = append(violations, artifactGCOverrideViolations(userSpec.ArtifactGC)...)
	if len(violations) > 0 {
		sort.Strings(violations)
		return fmt.Errorf("fields %v are not permitted when using workflowTemplateRef with templateReferencing restriction", violations)
	}
	return nil
}

// SanitizeUserWorkflowSpec returns a copy of userSpec with only allow-listed
// fields preserved. This provides defense-in-depth after validation.
func SanitizeUserWorkflowSpec(userSpec *wfv1.WorkflowSpec) *wfv1.WorkflowSpec {
	if userSpec == nil {
		return nil
	}
	sanitized := &wfv1.WorkflowSpec{}
	src := reflect.ValueOf(userSpec).Elem()
	dst := reflect.ValueOf(sanitized).Elem()
	t := src.Type()

	for i := 0; i < t.NumField(); i++ {
		if allowedUserOverrideFields[t.Field(i).Name] {
			dst.Field(i).Set(src.Field(i))
		}
	}
	// ArtifactGC is allow-listed wholesale above, which copies the pointer and
	// so would carry through the user's ServiceAccountName/PodSpecPatch/PodMetadata.
	// Strip those security-sensitive fields (on a copy, so the caller's spec is
	// left untouched) while preserving the benign ones.
	sanitized.ArtifactGC = sanitizeArtifactGC(sanitized.ArtifactGC)
	return sanitized
}

// artifactGCOverrideViolations reports the security-sensitive fields set within a
// user-supplied workflow-level ArtifactGC. These reach the artifact-GC Pod and
// must not be user-controllable when a hardened WorkflowTemplate is referenced
// under Strict/Secure mode.
func artifactGCOverrideViolations(agc *wfv1.WorkflowLevelArtifactGC) []string {
	if agc == nil {
		return nil
	}
	var violations []string
	if agc.PodSpecPatch != "" {
		violations = append(violations, "ArtifactGC.PodSpecPatch")
	}
	if agc.ServiceAccountName != "" {
		violations = append(violations, "ArtifactGC.ServiceAccountName")
	}
	if agc.PodMetadata != nil {
		violations = append(violations, "ArtifactGC.PodMetadata")
	}
	return violations
}

// sanitizeArtifactGC returns a deep copy of the workflow-level ArtifactGC with the
// security-sensitive override fields removed, leaving the benign fields (Strategy,
// ForceFinalizerRemoval) intact. A copy is returned so the caller's original spec
// is never mutated.
func sanitizeArtifactGC(agc *wfv1.WorkflowLevelArtifactGC) *wfv1.WorkflowLevelArtifactGC {
	if agc == nil {
		return nil
	}
	clean := agc.DeepCopy()
	clean.PodSpecPatch = ""
	clean.ServiceAccountName = ""
	clean.PodMetadata = nil
	return clean
}

// MergeTo will merge one workflow (the "patch" workflow) into another (the "target" workflow.
// If the target workflow defines a field, this take precedence over the patch.
func MergeTo(patch, target *wfv1.Workflow) error {
	if target == nil || patch == nil {
		return nil
	}

	// Temporarily remove hooks and labelsFrom as they don't merge
	patchHooks := patch.Spec.Hooks
	patch.Spec.Hooks = nil
	var patchLabelsFrom map[string]wfv1.LabelValueFrom
	if patch.Spec.WorkflowMetadata != nil {
		patchLabelsFrom = patch.Spec.WorkflowMetadata.LabelsFrom
		patch.Spec.WorkflowMetadata.LabelsFrom = nil
	}

	patchWfBytes, err := json.Marshal(patch)
	patch.Spec.Hooks = patchHooks
	if len(patchLabelsFrom) != 0 {
		patch.Spec.WorkflowMetadata.LabelsFrom = patchLabelsFrom
	}
	if err != nil {
		return err
	}

	targetWfByte, err := json.Marshal(target)
	if err != nil {
		return err
	}
	var mergedWfByte []byte

	mergedWfByte, err = strategicpatch.StrategicMergePatch(patchWfBytes, targetWfByte, wfv1.Workflow{})

	if err != nil {
		return err
	}

	target.Spec = wfv1.WorkflowSpec{}
	err = json.Unmarshal(mergedWfByte, target)
	if err != nil {
		return err
	}

	if len(patchHooks) != 0 && target.Spec.Hooks == nil {
		target.Spec.Hooks = make(wfv1.LifecycleHooks)
	}
	for name, hook := range patchHooks {
		// If the patch hook doesn't exist in target
		if _, ok := target.Spec.Hooks[name]; !ok {
			target.Spec.Hooks[name] = hook
		}
	}

	if len(patchLabelsFrom) != 0 && target.Spec.WorkflowMetadata.LabelsFrom == nil {
		target.Spec.WorkflowMetadata.LabelsFrom = make(map[string]wfv1.LabelValueFrom)
	}
	for key, val := range patchLabelsFrom {
		// If the patch labelFrom doesn't exist in target
		if _, ok := target.Spec.WorkflowMetadata.LabelsFrom[key]; !ok {
			target.Spec.WorkflowMetadata.LabelsFrom[key] = val
		}
	}
	return nil
}

// mergeMap will merge all element from right map to left map if it is not present in left.
func mergeMap(from, to map[string]string) {
	for key, val := range from {
		if _, ok := to[key]; !ok {
			to[key] = val
		}
	}
}

// JoinWorkflowMetaData will join the workflow metadata with the following order of preference
// 1. Workflow, 2 WorkflowTemplate (WorkflowTemplateRef), 3. WorkflowDefault.
func JoinWorkflowMetaData(wfMetaData, wfDefaultMetaData *metav1.ObjectMeta) {
	if wfDefaultMetaData != nil {
		mergeMetaDataTo(wfDefaultMetaData, wfMetaData)
	}
}

// JoinWorkflowSpec will join the workflow specs with the following order of preference
// 1. Workflow Spec, 2 WorkflowTemplate Spec (WorkflowTemplateRef), 3. WorkflowDefault Spec.
func JoinWorkflowSpec(wfSpec, wftSpec, wfDefaultSpec *wfv1.WorkflowSpec) (*wfv1.Workflow, error) {
	if wfSpec == nil {
		return nil, nil
	}
	targetWf := wfv1.Workflow{Spec: *wfSpec.DeepCopy()}
	if wftSpec != nil {
		err := MergeTo(&wfv1.Workflow{Spec: *wftSpec.DeepCopy()}, &targetWf)
		if err != nil {
			return nil, err
		}
	}
	if wfDefaultSpec != nil {
		err := MergeTo(&wfv1.Workflow{Spec: *wfDefaultSpec.DeepCopy()}, &targetWf)
		if err != nil {
			return nil, err
		}
	}

	// This condition will update the workflow Spec suspend value if merged value is different.
	// This scenario will happen when Workflow with WorkflowTemplateRef has suspend template
	if wfSpec.Suspend != targetWf.Spec.Suspend {
		targetWf.Spec.Suspend = wfSpec.Suspend
	}
	// The value of an argument should respect the priority order of the merge.
	// However, if Value, ValueFrom and Default are set in different places,
	// i.e. the Workflow, WorflowTemplate and WorkflowDefault, we should only keep the one with the greatest priority.
	wfParamsMap := parametersToMapByName(wfSpec)
	wftParamsMap := parametersToMapByName(wftSpec)
	for index, parameter := range targetWf.Spec.Arguments.Parameters {
		if parameter.HasValue() {
			if param, ok := wfParamsMap[parameter.Name]; ok {
				targetWf.Spec.Arguments.Parameters[index] = *param.DeepCopy()
			} else if param, ok := wftParamsMap[parameter.Name]; ok {
				targetWf.Spec.Arguments.Parameters[index] = *param.DeepCopy()
			}
			// If none of the above conditions are met, we can safely assume that the parameter is set from the wfDefaultSpec.
		}
	}

	// Apply the same priority logic for artifacts.
	// Artifacts from the workflow spec should take precedence over those from the template.
	wfArtifactsMap := artifactsToMapByName(wfSpec)
	wftArtifactsMap := artifactsToMapByName(wftSpec)
	for index, artifact := range targetWf.Spec.Arguments.Artifacts {
		if art, ok := wfArtifactsMap[artifact.Name]; ok {
			merged, err := mergeArtifactOverride(artifact, art)
			if err != nil {
				return nil, err
			}
			targetWf.Spec.Arguments.Artifacts[index] = merged
		} else if art, ok := wftArtifactsMap[artifact.Name]; ok {
			merged, err := mergeArtifactOverride(artifact, art)
			if err != nil {
				return nil, err
			}
			targetWf.Spec.Arguments.Artifacts[index] = merged
		}
		// If none of the above conditions are met, we can safely assume that the artifact is set from the wfDefaultSpec.
	}

	return &targetWf, nil
}

// mergeArtifactOverride applies override on top of base, giving override
// precedence. If both use the same storage backend (e.g. S3), fields not set
// on override (e.g. bucket/endpoint inherited from a lower-priority source)
// are preserved instead of being dropped by a full overwrite. If the backend
// differs (or either has no location configured), override fully replaces
// base's location, since merging fields from two different backend types
// would produce an invalid artifact with more than one location type set.
func mergeArtifactOverride(base, override wfv1.Artifact) (wfv1.Artifact, error) {
	baseType, baseErr := base.ArtifactLocation.Get()
	overrideType, overrideErr := override.ArtifactLocation.Get()
	if baseErr != nil || overrideErr != nil || reflect.TypeOf(baseType) != reflect.TypeOf(overrideType) {
		return *override.DeepCopy(), nil
	}

	baseJSON, err := json.Marshal(base.ArtifactLocation)
	if err != nil {
		return wfv1.Artifact{}, err
	}
	overrideJSON, err := json.Marshal(override.ArtifactLocation)
	if err != nil {
		return wfv1.Artifact{}, err
	}
	mergedLocationJSON, err := strategicpatch.StrategicMergePatch(baseJSON, overrideJSON, wfv1.ArtifactLocation{})
	if err != nil {
		return wfv1.Artifact{}, err
	}

	merged := *override.DeepCopy()
	merged.ArtifactLocation = wfv1.ArtifactLocation{}
	if err := json.Unmarshal(mergedLocationJSON, &merged.ArtifactLocation); err != nil {
		return wfv1.Artifact{}, err
	}
	return merged, nil
}

func parametersToMapByName(spec *wfv1.WorkflowSpec) map[string]wfv1.Parameter {
	parameterMap := make(map[string]wfv1.Parameter)
	if spec != nil {
		for _, param := range spec.Arguments.Parameters {
			if param.HasValue() {
				parameterMap[param.Name] = param
			}
		}
	}
	return parameterMap
}

func artifactsToMapByName(spec *wfv1.WorkflowSpec) map[string]wfv1.Artifact {
	artifactMap := make(map[string]wfv1.Artifact)
	if spec != nil {
		for _, artifact := range spec.Arguments.Artifacts {
			artifactMap[artifact.Name] = artifact
		}
	}
	return artifactMap
}

// mergeMetadata will merge the labels and annotations into the target metadata.
func mergeMetaDataTo(from, to *metav1.ObjectMeta) {
	if from == nil {
		return
	}
	if from.Labels != nil {
		if to.Labels == nil {
			to.Labels = make(map[string]string)
		}
		mergeMap(from.Labels, to.Labels)
	}
	if from.Annotations != nil {
		if to.Annotations == nil {
			to.Annotations = make(map[string]string)
		}
		mergeMap(from.Annotations, to.Annotations)
	}
}
