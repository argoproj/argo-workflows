package workflow

// Workflow constants
const (
	Group                            string = "argoproj.io"
	WorkflowKind                     string = "Workflow"
	WorkflowSingular                 string = "workflow"
	WorkflowPlural                   string = "workflows"
	WorkflowShortName                string = "wf"
	WorkflowFullName                 string = WorkflowPlural + "." + Group
	WorkflowTemplateKind             string = "WorkflowTemplate"
	WorkflowTemplateSingular         string = "workflowtemplate"
	WorkflowTemplatePlural           string = "workflowtemplates"
	WorkflowTemplateShortName        string = "wftmpl"
	WorkflowTemplateFullName         string = WorkflowTemplatePlural + "." + Group
	CronWorkflowKind                 string = "CronWorkflow"
	CronWorkflowSingular             string = "cronworkflow"
	CronWorkflowPlural               string = "cronworkflows"
	CronWorkflowShortName            string = "cronwf"
	CronWorkflowFullName             string = CronWorkflowPlural + "." + Group
	ClusterWorkflowTemplateKind      string = "ClusterWorkflowTemplate"
	ClusterWorkflowTemplateSingular  string = "clusterworkflowtemplate"
	ClusterWorkflowTemplatePlural    string = "clusterworkflowtemplates"
	ClusterWorkflowTemplateShortName string = "cwftmpl"
	ClusterWorkflowTemplateFullName  string = ClusterWorkflowTemplatePlural + "." + Group
)

type CRD struct {
	Kind, Singular, Plural, ShortName, FullName string
}

var CRDs = []CRD{
	{
		Kind:      ClusterWorkflowTemplateKind,
		Singular:  ClusterWorkflowTemplateSingular,
		Plural:    ClusterWorkflowTemplatePlural,
		ShortName: ClusterWorkflowTemplateShortName,
		FullName:  ClusterWorkflowTemplateFullName,
	},
	{
		Kind:      CronWorkflowKind,
		Singular:  CronWorkflowSingular,
		Plural:    CronWorkflowPlural,
		ShortName: CronWorkflowShortName,
		FullName:  CronWorkflowFullName,
	},
	{
		Kind:      WorkflowKind,
		Singular:  WorkflowSingular,
		Plural:    WorkflowPlural,
		ShortName: WorkflowShortName,
		FullName:  WorkflowFullName,
	},
	{
		Kind:      WorkflowTemplateKind,
		Singular:  WorkflowTemplateSingular,
		ShortName: WorkflowTemplateShortName,
		FullName:  WorkflowTemplateFullName,
	},
}
