package workflow

// Workflow constants
const (
	Group                            string = "argoproj.io"
	Version                          string = "v1alpha1"
	APIVersion                       string = Group + "/" + Version
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
	WorkflowEventBindingPlural       string = "workfloweventbindings"
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
	WorkflowEventBindingKind         string = "WorkflowEventBinding"
	WorkflowTaskSetKind              string = "WorkflowTaskSet"
	WorkflowTaskSetSingular          string = "workflowtaskset"
	WorkflowTaskSetPlural            string = "workflowtasksets"
	WorkflowTaskSetShortName         string = "wfts"
	WorkflowTaskSetFullName          string = WorkflowTaskSetPlural + "." + Group
	WorkflowTaskResultKind           string = "WorkflowTaskResult"
	WorkflowArtifactGCTaskKind       string = "WorkflowArtifactGCTask"
	WorkflowArtifactGCTaskSingular   string = "workflowartifactgctask"
	WorkflowArtifactGCTaskPlural     string = "workflowartifactgctasks"
	WorkflowArtifactGCTaskShortName  string = "wfat"
	WorkflowArtifactGCTaskFullName   string = WorkflowArtifactGCTaskPlural + "." + Group
)
