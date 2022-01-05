package transpiler

type CWLRequirements interface {
	isCWLRequirement()
}

type DockerRequirement struct {
	Class                 string
	DockerPull            *string
	DockerLoad            *string
	DockerFile            *string
	DockerImport          *string
	DockerImageId         *string
	DockerOutputDirectory *string
}

type SoftwarePackage struct {
	Package string
	Version []string
	Specs   []string
}

type SoftwareRequirement struct {
	Class    string // constant SoftwareRequirement
	Packages []SoftwarePackage
}

type LoadListingRequirement struct {
	Class       string // constant LoadListingRequirement
	LoadListing *LoadListingEnum
}

type Dirent struct {
	entry     CWLExpressionString
	entryName *CWLExpressionString
	writeable *bool
}

type InitialWorkDirRequirementListing interface {
	isInitialWorkDirRequirementListing()
}

type InitialWorkDirRequirement struct {
	Class   string // constant InitialWorkDirRequirement
	Listing InitialWorkDirRequirementListing
}

type InlineJavascriptRequirement struct {
	Class         string // constant InlineJavascriptRequirement
	ExpressionLib []string
}

type SchemaDefRequirementType interface {
	isSchemaDefRequirementType()
}

type SchemaDefRequirement struct {
	Class string // constant SchemaDefRequirement
	Types []SchemaDefRequirementType
}

type EnvironmentDef struct {
	EnvName  string
	EnvValue CWLExpressionString
}

type EnvVarRequirement struct {
	Class  string // constant EnvVarRequirement
	EnvDef []EnvironmentDef
}

type ShellCommandRequirement struct {
	Class string // constant ShellCommandRequirement
}

type WorkReuse struct {
	Class       string // constant WorkReuse
	enableReuse CWLExpressionBool
}

type NetworkAccess struct {
	Class         string // constant NetworkAccess
	NetworkAccess CWLExpressionBool
}

type InplaceUpdateRequirement struct {
	Class         string // constant InplaceUpdateRequirement
	InplaceUpdate Bool
}

type ToolTimeLimit struct {
	Class     string // constant ToolTimeLimit
	TimeLimit CWLExpressionInt
}

type ResourceRequirement struct {
	Class     string // constand ResourceRequirement
	CoresMin  *CWLExpressionNum
	CoresMax  *CWLExpressionNum
	RamMin    *CWLExpressionNum
	RamMax    *CWLExpressionNum
	TmpDirMin *CWLExpressionNum
	TmpDirMax *CWLExpressionNum
	OutdirMin *CWLExpressionNum
	OutdirMax *CWLExpressionNum
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

type CommandlineInputRecordFieldType interface {
	Flatten
	isCommandlineInputRecordFieldType()
}

func (_ CWLNull) isCommandlineInputRecordFieldType()                      {}
func (_ CWLBool) isCommandlineInputRecordFieldType()                      {}
func (_ CWLInt) isCommandlineInputRecordFieldType()                       {}
func (_ CWLLong) isCommandlineInputRecordFieldType()                      {}
func (_ CWLFloat) isCommandlineInputRecordFieldType()                     {}
func (_ CWLDouble) isCommandlineInputRecordFieldType()                    {}
func (_ CWLString) isCommandlineInputRecordFieldType()                    {}
func (_ CWLFile) isCommandlineInputRecordFieldType()                      {}
func (_ CWLDirectory) isCommandlineInputRecordFieldType()                 {}
func (_ CommandlineInputRecordSchema) isCommandlineInputRecordFieldType() {}
func (_ CommandlineInputArraySchema) isCommandlineInputRecordFieldType()  {}
func (_ CommandlineInputEnumSchema) isCommandlineInputRecordFieldType()   {}

type CommandlineInputRecordField struct {
	Name           string
	Type           []CommandlineInputRecordFieldType // len(1) represents scalar len > 1 represents array
	Doc            []string
	Label          *string
	SecondaryFiles []CWLSecondaryFileSchema
	Streamable     *bool
	Format         CWLFormat
	LoadContents   *bool
	LoadListing    LoadListingEnum
	InputBinding   *CommandlineBinding
}

type CommandlineInputArraySchemaType interface {
	isCommandlineInputArraySchemaType()
}

func (_ CWLNull) isCommandlineInputArraySchemaType() {}

type CommandlineInputArraySchema struct {
	Items        CommandlineInputArraySchemaType
	Type         string // MUST be array
	Label        *string
	Doc          []string
	Name         *string
	InputBinding *CommandlineBinding
}

type CommandlineInputEnumSchema struct {
	Symbols      []string
	Type         string // MUST BE enum, only a placeholder for type verification purposes
	Label        *string
	Doc          []string
	Name         *string
	InputBinding *CommandlineBinding
}

type CommandlineInputRecordSchema struct {
	Type         string // MUST BE "record"
	Fields       *[]CommandlineInputRecordField
	Label        *string
	Doc          *[]string
	Name         *string
	inputBinding *CommandlineBinding
}

type FlattenContext struct {
	contextStr *string
	m          map[string]FlatType
}

func NewFlattenContext() FlattenContext {
	fc := FlattenContext{
		contextStr: nil,
		m:          make(map[string]FlatType),
	}
	return fc
}

func (ctx FlattenContext) Copy() {
	newFc := FlattenContext{m: ctx.m}
	if ctx.contextStr != nil {
		newStr := *ctx.contextStr
		newFc.contextStr = &newStr
	}
}

func (ctx FlattenContext) GetFlatTypes() map[string]FlatType {
	return ctx.m
}

type Flatten interface {
	flatten(context *FlattenContext, name string)
}

func flattenMapFillerHelper(ty interface{}, context *FlattenContext, name string) {
	var markerString string
	if context.contextStr != nil {
		markerString = *context.contextStr + "." + name
	} else {
		markerString = name
	}
	context.m[markerString] = ty.(FlatType)
}

func (_ CWLNull) flatten(context *FlattenContext, name string) {
	flattenMapFillerHelper(CWLNull{}, context, name)
}
func (_ CWLBool) flatten(context *FlattenContext, name string) {
	flattenMapFillerHelper(CWLBool{}, context, name)
}
func (_ CWLInt) flatten(context *FlattenContext, name string) {
	flattenMapFillerHelper(CWLInt{}, context, name)
}
func (_ CWLLong) flatten(context *FlattenContext, name string) {
	flattenMapFillerHelper(CWLLong{}, context, name)
}
func (_ CWLFloat) flatten(context *FlattenContext, name string) {
	flattenMapFillerHelper(CWLFloat{}, context, name)
}
func (_ CWLDouble) flatten(context *FlattenContext, name string) {
	flattenMapFillerHelper(CWLDouble{}, context, name)
}
func (_ CWLString) flatten(context *FlattenContext, name string) {
	flattenMapFillerHelper(CWLString{}, context, name)
}
func (_ CWLFile) flatten(context *FlattenContext, name string) {
	flattenMapFillerHelper(CWLFile{}, context, name)
}
func (_ CWLDirectory) flatten(context *FlattenContext, name string) {
	flattenMapFillerHelper(CWLDirectory{}, context, name)
}
func (_ CWLStdin) flatten(context *FlattenContext, name string) {
	flattenMapFillerHelper(CWLStdin{}, context, name)
}

func bottom() {
	panic("Not yet implemented")
}

func (_ CommandlineInputRecordField) flatten(context *FlattenContext, name string) {
	bottom()
}

func (rschema CommandlineInputRecordSchema) flatten(context *FlattenContext, name string) {
	bottom()
}
func (_ CommandlineInputEnumSchema) flatten(context *FlattenContext, name string) {
	bottom()
}
func (_ CommandlineInputArraySchema) flatten(context *FlattenContext, name string) {
	bottom()
}

type FlatType interface {
	isFlat()
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

type CommandlineInputParameterType interface {
	Flatten
	isCLIParamType()
	toRecordFieldType() CommandlineInputRecordFieldType
}

func (_ CWLNull) isCLIParamType()                      {}
func (_ CWLBool) isCLIParamType()                      {}
func (_ CWLInt) isCLIParamType()                       {}
func (_ CWLLong) isCLIParamType()                      {}
func (_ CWLFloat) isCLIParamType()                     {}
func (_ CWLDouble) isCLIParamType()                    {}
func (_ CWLString) isCLIParamType()                    {}
func (_ CWLFile) isCLIParamType()                      {}
func (_ CWLDirectory) isCLIParamType()                 {}
func (_ CWLStdin) isCLIParamType()                     {}
func (_ CommandlineInputRecordSchema) isCLIParamType() {}
func (_ CommandlineInputEnumSchema) isCLIParamType()   {}
func (_ CommandlineInputArraySchema) isCLIParamType()  {}
func (_ String) isCLIParamType()                       {}

func (val CWLNull) toRecordFieldType() CommandlineInputRecordFieldType      { return &val }
func (val CWLBool) toRecordFieldType() CommandlineInputRecordFieldType      { return &val }
func (val CWLInt) toRecordFieldType() CommandlineInputRecordFieldType       { return &val }
func (val CWLLong) toRecordFieldType() CommandlineInputRecordFieldType      { return &val }
func (val CWLFloat) toRecordFieldType() CommandlineInputRecordFieldType     { return &val }
func (val CWLDouble) toRecordFieldType() CommandlineInputRecordFieldType    { return &val }
func (val CWLString) toRecordFieldType() CommandlineInputRecordFieldType    { return &val }
func (val CWLFile) toRecordFieldType() CommandlineInputRecordFieldType      { return &val }
func (val CWLDirectory) toRecordFieldType() CommandlineInputRecordFieldType { return &val }
func (val CWLStdin) toRecordFieldType() CommandlineInputRecordFieldType     { return nil }
func (val CommandlineInputRecordSchema) toRecordFieldType() CommandlineInputRecordFieldType {
	return &val
}

func (val CommandlineInputEnumSchema) toRecordFieldType() CommandlineInputRecordFieldType {
	return &val
}

func (val CommandlineInputArraySchema) toRecordFieldType() CommandlineInputRecordFieldType {
	return &val
}

func (val String) toRecordFieldType() CommandlineInputRecordFieldType { return nil }

type CommandlineBinding struct {
	LoadContents  *bool
	Position      *int
	Prefix        *string
	Seperate      *bool
	ItemSeperator *string
	ValueFrom     CWLExpressionString
	ShellQuote    *bool
}

type CommandlineInputParameter struct {
	Type           []CommandlineInputParameterType // len(1) == scalar while len > 1 == array
	Label          *string
	SecondaryFiles []CWLSecondaryFileSchema // len(1) == scalar while len > 1 == array
	Streamable     *bool
	Doc            []string
	Id             *string
	Format         CWLFormat
	LoadContents   *bool
	LoadListing    LoadListingEnum
	Default        *interface{}
	InputBinding   *CommandlineBinding
}

type CommandlineOutputBindingGlob interface {
	isCommandlineOutputBindingGlob()
}

func (_ String) isCommandlineOutputBindingGlob()        {}
func (_ Strings) isCommandlineOutputBindingGlob()       {}
func (_ CWLExpression) isCommandlineOutputBindingGlob() {}

type CommandlineOutputBinding struct {
	LoadContents *bool
	LoadListing  LoadListingEnum
	Glob         CommandlineOutputBindingGlob
	OutputEval   CWLExpression
}

type CommandlineOutputParameterType interface {
	isCommandlineOutputParameterType()
}

type CommandlineOutputParameter struct {
	Type           []CommandlineOutputParameterType
	Label          *string
	SecondaryFiles []CWLSecondaryFileSchema
	Streamable     *bool
	Doc            []string
	Id             *string
	Format         CWLFormat
	OutputBinding  *CommandlineOutputBinding
}

type CommandlineArgument interface {
	isCommandlineArguement()
}

func (_ String) isCommandlineArguement()             {}
func (_ CWLExpression) isCommandlineArguement()      {}
func (_ CommandlineBinding) isCommandlineArguement() {}

type CommandlineTool struct {
	Inputs       []CommandlineInputParameter
	Outputs      []CommandlineOutputParameter
	Class        string // Must be "CommandLineTool"
	Id           *string
	Label        *string
	Doc          []string
	Requirements []CWLRequirements
	Hints        []interface{}
	CWLVersion   *string
	Intent       []string
	BaseCommand  []string
	Arguments    []CommandlineArgument
	Stdin        *CWLExpressionString
	Stderr       *CWLExpressionString
	Stdout       *CWLExpressionString
}
