package transpiler

// interface due to large number of child types
type CWLRequirements interface {
	isCWLRequirement()
}

type DockerRequirement struct {
	Class                 string  `yaml:"class"`
	DockerPull            *string `yaml:"dockerPull"`
	DockerLoad            *string `yaml:"dockerLoad"`
	DockerFile            *string `yaml:"dockerFile"`
	DockerImport          *string `yaml:"dockerImport"`
	DockerImageId         *string `yaml:"dockerImageId"`
	DockerOutputDirectory *string `yaml:"dockerOutputDirectory"`
}

type SoftwarePackage struct {
	Package string  `yaml:"package"`
	Version Strings `yaml:"version"`
	Specs   Strings `yaml:"specs"`
}

type SoftwareRequirement struct {
	Class    string            `yaml:"class"` // constant SoftwareRequirement
	Packages []SoftwarePackage `yaml:"packages"`
}

type LoadListingRequirement struct {
	Class       string           `yaml:"class"` // constant LoadListingRequirement
	LoadListing *LoadListingEnum `yaml:"loadListing"`
}

type Dirent struct {
	entry     CWLExpression  `yaml:"entry"`
	entryName *CWLExpression `yaml:"entryName"`
	writeable *bool          `yaml:"writeable"`
}

type InitialWorkDirRequirementListing interface {
	isInitialWorkDirRequirementListing()
}

type InitialWorkDirRequirement struct {
	Class   string                           `yaml:"class"` // constant InitialWorkDirRequirement
	Listing InitialWorkDirRequirementListing `yaml:"listing"`
}

type InlineJavascriptRequirement struct {
	Class         string  `yaml:"class"` // constant InlineJavascriptRequirement
	ExpressionLib Strings `yaml:"expressionLib"`
}

type SchemaDefRequirementType interface {
	isSchemaDefRequirementType()
}

type SchemaDefRequirement struct {
	Class string                     `yaml:"class"` // constant SchemaDefRequirement
	Types []SchemaDefRequirementType `yaml:"types"`
}

type EnvironmentDef struct {
	EnvName  string        `yaml:"envName"`
	EnvValue CWLExpression `yaml:"envValue"`
}

type EnvVarRequirement struct {
	Class  string           `yaml:"class"` // constant EnvVarRequirement
	EnvDef []EnvironmentDef `yaml:"envDef"`
}

type ShellCommandRequirement struct {
	Class string `yaml:"class"` // constant ShellCommandRequirement
}

type WorkReuse struct {
	Class       string        `yaml:"class"` // constant WorkReuse
	EnableReuse CWLExpression `yaml:"enableReuse"`
}

type NetworkAccess struct {
	Class         string // constant NetworkAccess
	NetworkAccess CWLExpression
}

type InplaceUpdateRequirement struct {
	Class         string `yaml:"class"` // constant InplaceUpdateRequirement
	InplaceUpdate Bool   `yaml:"inplaceUpdate"`
}

type ToolTimeLimit struct {
	Class     string        `yaml:"class"` // constant ToolTimeLimit
	TimeLimit CWLExpression `yaml:"timeLimit"`
}

type ResourceRequirement struct {
	Class     string         `yaml:"class"` // constand ResourceRequirement
	CoresMin  *CWLExpression `yaml:"coresMin"`
	CoresMax  *CWLExpression `yaml:"coresMax"`
	RamMin    *CWLExpression `yaml:"ramMin"`
	RamMax    *CWLExpression `yaml:"ramMax"`
	TmpdirMin *CWLExpression `yaml:"tmpdirMin"`
	TmpdirMax *CWLExpression `yaml:"tmpdirMin"`
	OutdirMin *CWLExpression `yaml:"outdirMin"`
	OutdirMax *CWLExpression `yaml:"outdirMax"`
}

func (_ InlineJavascriptRequirement) isCWLRequirement() {}
func (_ SchemaDefRequirement) isCWLRequirement()        {}
func (_ LoadListingRequirement) isCWLRequirement()      {}
func (_ DockerRequirement) isCWLRequirement()           {}
func (_ SoftwareRequirement) isCWLRequirement()         {}
func (_ InitialWorkDirRequirement) isCWLRequirement()   {}
func (_ EnvVarRequirement) isCWLRequirement()           {}
func (_ ShellCommandRequirement) isCWLRequirement()     {}
func (_ WorkReuse) isCWLRequirement()                   {}
func (_ NetworkAccess) isCWLRequirement()               {}
func (_ InplaceUpdateRequirement) isCWLRequirement()    {}
func (_ ToolTimeLimit) isCWLRequirement()               {}

func (_ CommandlineInputRecordSchema) isSchemaDefRequirementType() {}
func (_ CommandlineInputEnumSchema) isSchemaDefRequirementType()   {}
func (_ CommandlineInputArraySchema) isSchemaDefRequirementType()  {}
func (_ DockerRequirement) isSchemaDefRequirementType()            {}
func (_ SoftwareRequirement) isSchemaDefRequirementType()          {}
func (_ InitialWorkDirRequirement) isSchemaDefRequirementType()    {}

type CommandlineInputRecordField struct {
	Name           string              `yaml:"name"`
	Type           CommandlineTypes    `yaml:"type"` // len(1) represents scalar len > 1 represents array
	Doc            Strings             `yaml:"doc"`
	Label          *string             `yaml:"label"`
	SecondaryFiles SecondaryFiles      `yaml:"secondaryFiles"`
	Streamable     *bool               `yaml:"streamable"`
	Format         CWLFormat           `yaml:"format"`
	LoadContents   *bool               `yaml:"loadContents"`
	LoadListing    LoadListingEnum     `yaml:"loadListing"`
	InputBinding   *CommandlineBinding `yaml:"inputBinding"`
}

type CommandlineInputArraySchema struct {
	Items        CommandlineTypes    `yaml:"items"`
	Type         string              `yaml:"type"` // MUST be array
	Label        *string             `yaml:"label"`
	Doc          Strings             `yaml:"doc"`
	Name         *string             `yaml:"name"`
	InputBinding *CommandlineBinding `yaml:"inputBinding"`
}

type CommandlineInputEnumSchema struct {
	Symbols      Strings `yaml:"symbols"`
	Type         string  `yaml:"type"` // MUST BE enum, only a placeholder for type verification purposes
	Label        *string `yaml:"label"`
	Doc          Strings `yaml:"doc"`
	Name         *string `yaml:"name"`
	InputBinding *CommandlineBinding
}

type CommandlineInputRecordSchema struct {
	Type         string                         `yaml:"type"` // MUST BE "record"
	Fields       *[]CommandlineInputRecordField `yaml:"fields"`
	Label        *string                        `yaml:"label"`
	Doc          *Strings                       `yaml:"doc"`
	Name         *string                        `yaml:"name"`
	inputBinding *CommandlineBinding            `yaml:"inputBinding"`
}

func (_ CWLNull) isFlat()                    {}
func (_ CWLBool) isFlat()                    {}
func (_ CWLInt) isFlat()                     {}
func (_ CWLLong) isFlat()                    {}
func (_ CWLFloat) isFlat()                   {}
func (_ CWLDouble) isFlat()                  {}
func (_ CWLFile) isFlat()                    {}
func (_ CWLDirectory) isFlat()               {}
func (_ CWLStdin) isFlat()                   {}
func (_ String) isFlat()                     {}
func (_ CommandlineInputEnumSchema) isFlat() {}

type Type int32

const (
	CWLNullKind Type = iota
	CWLBoolKind
	CWLIntKind
	CWLLongKind
	CWLFloatKind
	CWLDoubleKind
	CWLFileKind
	CWLDirectoryKind
	CWLStdinKind
	CWLStringKind
	CWLRecordKind
	CWLRecordFieldKind
	CWLEnumKind
	CWLArrayKind
)

type CommandlineType struct {
	Kind   Type
	Record *CommandlineInputRecordSchema
	Enum   *CommandlineInputEnumSchema
	Array  *CommandlineInputArraySchema
}

type CommandlineTypes []CommandlineType

type CommandlineBinding struct {
	LoadContents  *bool         `yaml:"loadContents"`
	Position      *int          `yaml:"position"`
	Prefix        *string       `yaml:"prefix"`
	Seperate      *bool         `yaml:"seperate"`
	ItemSeperator *string       `yaml:"itemSeperator"`
	ValueFrom     CWLExpression `yaml:"valueFrom"`
	ShellQuote    *bool         `yaml:"bool"`
}

type CommandlineInputParameter struct {
	Type           CommandlineTypes    `yaml:"type"` // len(1) == scalar while len > 1 == array
	Label          *string             `yaml:"label"`
	SecondaryFiles SecondaryFiles      `yaml:"secondaryFiles"` // len(1) == scalar while len > 1 == array
	Streamable     *bool               `yaml:"streamable"`
	Doc            Strings             `yaml:"doc"`
	Id             *string             `yaml:"id"`
	Format         *CWLFormat          `yaml:"format"`
	LoadContents   *bool               `yaml:"loadContents"`
	LoadListing    *LoadListingEnum    `yaml:"loadListing"`
	Default        interface{}         `yaml:"default"`
	InputBinding   *CommandlineBinding `yaml:"inputBinding"`
}

type OutputBindingGlobKind int32

const (
	GlobStringKind OutputBindingGlobKind = iota
	GlobStringsKind
	GlobExpressionKind
)

type CommandlineOutputBindingGlob struct {
	Kind       OutputBindingGlobKind
	String     String
	Strings    Strings
	Expression CWLExpression
}

type CommandlineOutputBinding struct {
	LoadContents *bool                        `yaml:"loadContents"`
	LoadListing  LoadListingEnum              `yaml:"loadListing"`
	Glob         CommandlineOutputBindingGlob `yaml:"glob"`
	OutputEval   CWLExpression                `yaml:"outputEval"`
}

type CommandlineOutputParameter struct {
	Type           CommandlineTypes          `yaml:"type"`
	Label          *string                   `yaml:"label"`
	SecondaryFiles SecondaryFiles            `yaml:"secondaryFiles"`
	Streamable     *bool                     `yaml:"streamable"`
	Doc            Strings                   `yaml:"doc"`
	Id             *string                   `yaml:"id"`
	Format         *CWLFormat                `yaml:"format"`
	OutputBinding  *CommandlineOutputBinding `yaml:"outputBinding"`
}

type CommandlineArgumentKind int32

const (
	ArgumentStringKind CommandlineArgumentKind = iota
	ArgumentExpressionKind
	ArgumentCLIBindingKind
)

type CommandlineArgument struct {
	Kind               CommandlineArgumentKind
	String             String
	Expression         CWLExpression
	CommandlineBinding CommandlineBinding
}

type Inputs []CommandlineInputParameter
type Outputs []CommandlineOutputParameter
type Requirements []CWLRequirements
type Hints []interface{}
type Arguments []CommandlineArgument

type CommandlineTool struct {
	Inputs       Inputs         `yaml:"inputs"`
	Outputs      Outputs        `yaml:"outputs"`
	Class        string         `yaml:"class"` // Must be "CommandLineTool"
	Id           *string        `yaml:"id"`
	Label        *string        `yaml:"label"`
	Doc          Strings        `yaml:"doc"`
	Requirements Requirements   `yaml:"requirements"`
	Hints        Hints          `yaml:"hints"`
	CWLVersion   *string        `yaml:"cwlVersion"`
	Intent       Strings        `yaml:"intent"`
	BaseCommand  Strings        `yaml:"baseCommand"`
	Arguments    Arguments      `yaml:"arguments"`
	Stdin        *CWLExpression `yaml:"stdin"`
	Stderr       *CWLExpression `yaml:"stderr"`
	Stdout       *CWLExpression `yaml:"stdout"`
}
