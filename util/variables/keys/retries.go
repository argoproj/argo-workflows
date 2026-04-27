package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// ─────────────────────────── retries.* — inside retry-strategy templates

var (
	Retries = v.Define(v.Spec{
		Template:    "retries",
		Kind:        v.KindRetry,
		ValueType:   "string",
		AppliesTo:   v.PodKinds,
		Phases:      []v.LifecyclePhase{v.PhInsideRetry, v.PhDuringExecute},
		Description: "0-based retry attempt index",
	})
	RetriesLastExitCode = v.Define(v.Spec{
		Template:    "retries.lastExitCode",
		Kind:        v.KindRetry,
		ValueType:   "string",
		AppliesTo:   v.PodKinds,
		Phases:      []v.LifecyclePhase{v.PhInsideRetry, v.PhDuringExecute},
		Description: "Exit code of the previous attempt (or 0 on first attempt)",
	})
	RetriesLastStatus = v.Define(v.Spec{
		Template:    "retries.lastStatus",
		Kind:        v.KindRetry,
		ValueType:   "string",
		AppliesTo:   v.PodKinds,
		Phases:      []v.LifecyclePhase{v.PhInsideRetry, v.PhDuringExecute},
		Description: "Phase of the previous attempt (or empty on first)",
	})
	RetriesLastDuration = v.Define(v.Spec{
		Template:    "retries.lastDuration",
		Kind:        v.KindRetry,
		ValueType:   "string",
		AppliesTo:   v.PodKinds,
		Phases:      []v.LifecyclePhase{v.PhInsideRetry, v.PhDuringExecute},
		Description: "Duration of the previous attempt in seconds",
	})
	RetriesLastMessage = v.Define(v.Spec{
		Template:    "retries.lastMessage",
		Kind:        v.KindRetry,
		ValueType:   "string",
		AppliesTo:   v.PodKinds,
		Phases:      []v.LifecyclePhase{v.PhInsideRetry, v.PhDuringExecute},
		Description: "Message of the previous attempt",
	})
)
