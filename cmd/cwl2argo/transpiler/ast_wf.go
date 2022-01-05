package transpiler

type InputEnumSchema struct {
	Symbols []string
	Type    string //constant enum
	Label   *string
	Doc     *[]string
	Name    *string
}

type InputArraySchema struct {
	Items []SharedWorkflowInputSumTypes
	Type  string //constant array
	Label *string
	Doc   *[]string
	Name  string
}

type InputRecordField struct {
	Name           string
	Type           []SharedWorkflowInputSumTypes
	Doc            *[]string
	Label          *string
	SecondaryFiles *CWLSecondaryFileSchema
	Streamable     *bool
	Format         *CWLFormat
	LoadContents   *bool
	LoadListing    *LoadListingEnum
}

type InputRecordSchema struct {
	Type   string // constant record
	Fields *[]InputRecordField
	Label  *string
	Doc    *[]string
	Name   *string
}

type SharedWorkflowInputSumTypes interface {
	isSharedWorkflowInputSumType()
}

func (_ CWLNull) isSharedWorkflowInputSumType()           {}
func (_ CWLBool) isSharedWorkflowInputSumType()           {}
func (_ CWLInt) isSharedWorkflowInputSumType()            {}
func (_ CWLLong) isSharedWorkflowInputSumType()           {}
func (_ CWLFloat) isSharedWorkflowInputSumType()          {}
func (_ CWLDouble) isSharedWorkflowInputSumType()         {}
func (_ CWLString) isSharedWorkflowInputSumType()         {}
func (_ CWLFile) isSharedWorkflowInputSumType()           {}
func (_ CWLDirectory) isSharedWorkflowInputSumType()      {}
func (_ InputRecordSchema) isSharedWorkflowInputSumType() {}
func (_ InputEnumSchema) isSharedWorkflowInputSumType()   {}
func (_ InputArraySchema) isSharedWorkflowInputSumType()  {}
func (_ String) isSharedWorkflowInputSumType()            {}

type InputBinding struct {
	LoadContents *bool
}

type WorkflowInputParameter struct {
	Type           SharedWorkflowInputSumTypes
	Label          *string
	SecondaryFiles *[]CWLSecondaryFileSchema
	Streamable     *bool
	Doc            *[]string
	Id             *string
	Format         *CWLFormat
	LoadContents   *bool
	LoadListing    LoadListingEnum
	Default        interface{}
	InputBinding   InputBinding
}

type WorkflowOutputParameterType interface{}

type LinkMergeMethod interface {
	isLinkMergeMethod()
}
type MergeNested struct{}
type MergeFlattened struct{}

func (_ MergeNested) isLinkMergeMethod()    {}
func (_ MergeFlattened) isLinkMergeMethod() {}

type PickValueMethod interface {
	isPickValueMethod()
}
type FirstNonNull struct{}
type TheOnlyNonNull struct{}
type AllNonNull struct{}

func (_ FirstNonNull) isPickValueMethod()   {}
func (_ TheOnlyNonNull) isPickValueMethod() {}
func (_ AllNonNull) isPickValueMethod()     {}

type SharedWorkflowOutputSumTypes interface {
	isSharedWorkflowOutputSumType()
}

type OutputRecordFields struct {
	Name           string
	Type           SharedWorkflowOutputSumTypes
	Doc            []string
	Label          *string
	SecondaryFiles []CWLSecondaryFileSchema
	Streamable     *bool
	Format         *CWLFormat
}

type OutputRecordSchema struct {
	Type   string //constant record
	Fields []OutputRecordFields
	Label  *string
	Doc    []string
	Name   *string
}

type WorkflowOutputParameter struct {
	Type           WorkflowOutputParameterType
	Label          *string
	SecondaryFiles []CWLSecondaryFileSchema
	Streamable     *bool
	Doc            []string
	Id             *string
	Format         *CWLFormat
	OutputSource   []string
	LinkMerge      *LinkMergeMethod
	PickValue      *PickValueMethod
}

type WorkflowStepInput struct {
	Id           *string
	Source       *string
	LinkMerge    *LinkMergeMethod
	PickValue    *PickValueMethod
	LoadContents *bool
	LoadListing  *LoadListingEnum
	Label        *string
	Default      *interface{}
	ValueFrom    *CWLExpressionString
}

type WorkflowStepOutput struct {
	Id *string
}

type WorkflowRunnable interface {
	isWorkflowRunnable()
}

func (_ CommandlineTool) isWorkflowRunnable() {}
func (_ String) isWorkflowRunnable()          {}
func (_ Workflow) isWorkflowRunnable()        {}

type WorkflowRequirements interface {
	isWorkflowRequirement()
}

type SubworkflowFeatureRequirement struct {
	Class string // constant SubworkflowFeatureRequirement
}
type ScatterFeatureRequirement struct {
	Class string // constant ScatterFeatureRequirement
}
type MultipleInputFeatureRequirement struct {
	Class string // constant MultipleInputFeatureRequirement
}

type StepInputExpressionRequirement struct {
	Class string // constant StepInputExpressionRequirement
}

func (_ InlineJavascriptRequirement) isWorkflowRequirement()     {}
func (_ SchemaDefRequirement) isWorkflowRequirement()            {}
func (_ LoadListingRequirement) isWorkflowRequirement()          {}
func (_ DockerRequirement) isWorkflowRequirement()               {}
func (_ SoftwareRequirement) isWorkflowRequirement()             {}
func (_ InitialWorkDirRequirement) isWorkflowRequirement()       {}
func (_ EnvVarRequirement) isWorkflowRequirement()               {}
func (_ ShellCommandRequirement) isWorkflowRequirement()         {}
func (_ WorkReuse) isWorkflowRequirement()                       {}
func (_ NetworkAccess) isWorkflowRequirement()                   {}
func (_ InplaceUpdateRequirement) isWorkflowRequirement()        {}
func (_ ToolTimeLimit) isWorkflowRequirement()                   {}
func (_ SubworkflowFeatureRequirement) isWorkflowRequirement()   {}
func (_ ScatterFeatureRequirement) isWorkflowRequirement()       {}
func (_ MultipleInputFeatureRequirement) isWorkflowRequirement() {}
func (_ StepInputExpressionRequirement) isWorkflowRequirement()  {}

type ScatterMethod interface {
	isScatterMethod()
}
type DotProduct struct{}
type NestedCrossProduct struct{}
type FlatCrossProduct struct{}

func (_ DotProduct) isScatterMethod()         {}
func (_ NestedCrossProduct) isScatterMethod() {}
func (_ FlatCrossProduct) isScatterMethod()   {}

type WorkflowStep struct {
	In            []WorkflowStepInput
	Out           []WorkflowStepOutput
	Run           WorkflowRunnable
	Id            *string
	Label         *string
	Doc           []string
	Requirements  []WorkflowRequirements
	Hints         []interface{}
	When          *CWLExpression
	Scatter       []string
	ScatterMethod ScatterMethod
}

type Workflow struct {
	Inputs       []WorkflowInputParameter
	Outputs      []WorkflowOutputParameter
	Class        string // constant Workflow
	Steps        []WorkflowStep
	Id           *string
	Label        *string
	Doc          []string
	Requirements []CWLRequirements
	Hints        []interface{}
	CWLVersion   *string
	Intent       []string
}
