package parameterization

import p "github.com/argoproj/argo-workflows/v4/util/parameterization"

// CapabilityReport describes implemented boundaries, not a promise that an
// arbitrary workflow is retryable. Preview checks its concrete frozen graph.
type CapabilityReport struct {
	Experimental         bool     `json:"experimental"`
	SchemaDialect        string   `json:"schemaDialect"`
	NativeWorkflowOwner  string   `json:"nativeWorkflowOwner"`
	Features             []string `json:"features"`
	Unsupported          []string `json:"unsupported"`
	EncryptedSecretInput bool     `json:"encryptedSecretInput"`
	ActionCallScopes     []string `json:"actionCallScopes"`
	SecretVersionPolicy  string   `json:"secretVersionPolicy"`
	PartialRetryScopes   []string `json:"partialRetryScopes"`
}

func capabilities(encryptedInput bool) CapabilityReport {
	c := CapabilityReport{
		Experimental: true, SchemaDialect: p.Dialect, NativeWorkflowOwner: "Argo",
		Features:             []string{"structuredParameters", "saveRevision", "history", "semanticDiff", "executionDiff", "preview", "submit", "actionBindings", "originalParameterRetry", "secretReferences", "dynamicOptions", "derivedValues", "intermediateInputForms", "staticTemplateRefs", "clusterTemplateBindings", "checkedPartialRetry", "definitionPublications", "definitionDiff", "namedActionCalls", "actionParameterMappings", "latestSecretVersionIndex", "actionArtifactMappings", "externalVaultVersionMaterialization", "approvedLoopTopologyRetry"},
		Unsupported:          []string{"dynamicTemplateRef", "actionCallPathThroughLoop", "partialRetryTemplateTopologyChanges", "partialRetryPerItemLoopReuse", "partialRetryUnverifiedArtifactReuse", "partialRetryArtifactRepositoryDefaults", "partialRetryCompositeArtifactForwarding", "partialRetryMemoization", "externalSecretDirectInjection", "perAttemptSecretLease"},
		EncryptedSecretInput: encryptedInput,
		ActionCallScopes:     []string{"namedDAGCalls", "namedStepsCalls", "staticNestedCalls", "terminalLoopCall"},
		SecretVersionPolicy:  "resolve-latest-at-reviewed-submit; original-retry-retains-pinned-version",
		PartialRetryScopes:   []string{"dag", "steps", "staticNestedDAGAndSteps", "stableIdentityCompositeLoops", "replaceApprovedLoopExpansion", "nativeLeafRetryGroups", "httpArtifactChecksumReuse"},
	}
	if !encryptedInput {
		c.Unsupported = append(c.Unsupported, "encryptedInput")
	}
	c.Features = append(c.Features, "authorReviewedActionReplacement", "nativeLeafRetryGroups", "httpArtifactChecksumReuse", "taskPodExternalSecretDispatchCheck")
	c.Features = append(c.Features, "callSiteResultBindings", "frozenResultContracts", "ownedResourceObservations", "declarativeResultViews")
	c.Unsupported = append(c.Unsupported, "automaticActionEffectProof", "legacyActionSemanticVerification")
	return c
}
