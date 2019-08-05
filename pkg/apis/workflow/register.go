package workflow

// Workflow constants
const (
	Group                     string = "argoproj.io"
	WorkflowKind              string = "Workflow"
	WorkflowSingular          string = "workflow"
	WorkflowPlural            string = "workflows"
	WorkflowShortName         string = "wf"
	WorkflowFullName          string = WorkflowPlural + "." + Group
	WorkflowTemplateKind      string = "WorkflowTemplate"
	WorkflowTemplateSingular  string = "workflowtemplate"
	WorkflowTemplatePlural    string = "workflowtemplates"
	WorkflowTemplateShortName string = "wftmpl"
	WorkflowTemplateFullName  string = WorkflowTemplatePlural + "." + Group
)
