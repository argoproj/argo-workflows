package workflow

// Workflow constants
const (
	Group                            string = "argoproj.io"
	WorkflowKind                            = "Workflow"
	WorkflowSingular                        = "workflow"
	WorkflowPlural                          = "workflows"
	WorkflowShortName                       = "wf"
	WorkflowFullName                        = WorkflowPlural + "." + Group
	WorkflowTemplateKind                    = "WorkflowTemplate"
	WorkflowTemplateSingular                = "workflowtemplate"
	WorkflowTemplatePlural                  = "workflowtemplates"
	WorkflowTemplateShortName               = "wftmpl"
	WorkflowTemplateFullName                = WorkflowTemplatePlural + "." + Group
	CronWorkflowKind                        = "CronWorkflow"
	CronWorkflowSingular                    = "cronworkflow"
	CronWorkflowPlural                      = "cronworkflows"
	CronWorkflowShortName                   = "cronwf"
	CronWorkflowFullName                    = CronWorkflowPlural + "." + Group
	ClusterWorkflowTemplateKind             = "ClusterWorkflowTemplate"
	ClusterWorkflowTemplateSingular         = "clusterworkflowtemplate"
	ClusterWorkflowTemplatePlural           = "clusterworkflowtemplates"
	ClusterWorkflowTemplateShortName        = "cwftmpl"
	ClusterWorkflowTemplateFullName         = ClusterWorkflowTemplatePlural + "." + Group
)

type ScopeType = string

const (
	Cluster    ScopeType = "Cluster"
	Namespaced           = "Namespaced"
)

type CRD struct {
	Kind, Singular, Plural, ShortName, FullName string
	Scope                                       ScopeType
}

var CRDs = []CRD{
	{
		Kind:      ClusterWorkflowTemplateKind,
		Singular:  ClusterWorkflowTemplateSingular,
		Plural:    ClusterWorkflowTemplatePlural,
		ShortName: ClusterWorkflowTemplateShortName,
		FullName:  ClusterWorkflowTemplateFullName,
		Scope:     Cluster,
	},
	{
		Kind:      CronWorkflowKind,
		Singular:  CronWorkflowSingular,
		Plural:    CronWorkflowPlural,
		ShortName: CronWorkflowShortName,
		FullName:  CronWorkflowFullName,
		Scope:     Namespaced,
	},
	{
		Kind:      WorkflowKind,
		Singular:  WorkflowSingular,
		Plural:    WorkflowPlural,
		ShortName: WorkflowShortName,
		FullName:  WorkflowFullName,
		Scope:     Namespaced,
	},
	{
		Kind:      WorkflowTemplateKind,
		Singular:  WorkflowTemplateSingular,
		Plural:    WorkflowTemplatePlural,
		ShortName: WorkflowTemplateShortName,
		FullName:  WorkflowTemplateFullName,
		Scope:     Namespaced,
	},
}
