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

type ScopeType = string

const (
	Cluster    ScopeType = "Cluster"
	Namespaced ScopeType = "Namespaced"
)

type CRD struct {
	Kind, Singular, Plural, ShortName, FullName string
	Scope                                       ScopeType
	AdditionalPrinterColumns                    []PrinterColumn
}

type PrinterColumn struct {
	JSONPath    string `json:"jsonPath"`
	Description string `json:"description"`
	Format      string `json:"format"`
	Name        string `json:"name"`
	Type        string `json:"type"`
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
		AdditionalPrinterColumns: []PrinterColumn{
			{

				JSONPath:    ".status.phase",
				Description: "Status of the workflow",
				Name:        "Status",
				Type:        "string",
			}, {
				JSONPath:    ".status.startedAt",
				Description: "When the workflow was started",
				Format:      "date-time",
				Name:        "Age",
				Type:        "date",
			},
		},
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
