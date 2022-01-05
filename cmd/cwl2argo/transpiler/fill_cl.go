package transpiler

import (
	"errors"
	"fmt"
)

const (
	CommandlineToolClass = "CommandLineTool"
)

/* func getCLIArraySchema(aschema map[string]interface{}) (*CommandlineInputArraySchema, error) {
	var arraySchema CommandlineInputArraySchema

	typeVal, ok := aschema["type"]
	if !ok {
		return nil, errors.New("<type> expected in definition of ArraySchema")
	}

	if typeVal != "array" {
		return nil, errors.New("<type> expected \"array\"")
	}

	itemsi, ok := aschema["items"]
	if !ok {
		return nil, errors.New("<items> expected in definition of ArraySchema")
	}

	labeli, ok := aschema["label"]
	if ok {
	}

	doc, ok := aschema["doc"]
	if ok {
	}

	name, ok := aschema["name"]
	if ok {
	}

	inputBindingi, ok := aschema["inputBinding"]
	if ok {
	}

	return &arraySchema, nil
} */

func getCLIRecordType(typeValue interface{}) ([]CommandlineInputRecordFieldType, error) {
	tys := make([]CommandlineInputRecordFieldType, 0)

	// a hack to reduce code needed
	// the normal CLIType is a superset of the RecordTypes
	// therefore we can use this functionality to help implement filling record types
	ntys, err := getCLIType(typeValue)
	if err != nil {
		return nil, err
	}

	for _, ntyi := range ntys {
		ty := ntyi.toRecordFieldType()
		if ty == nil {
			return nil, fmt.Errorf("%T is not accepted", ty)
		}
		tys = append(tys, ty)
	}

	return tys, nil
}

func getCWLSecondaryFileSchemaString(secondaryFileStr string) CWLSecondaryFileSchema {
	var secondaryFile CWLSecondaryFileSchema
	secondaryFile.Pattern = CWLExpression{Expression: ""}
	secondaryFile.Required = nil

	if len(secondaryFileStr) > 1 {
		if secondaryFileStr[len(secondaryFileStr)-1] == '?' {
			secondaryFile.Pattern = String(secondaryFileStr[:len(secondaryFileStr)-1])
		}
	}

	return secondaryFile
}

func getCWLSecondaryFileSchemaMap(secondaryFileMap map[string]interface{}) (*CWLSecondaryFileSchema, error) {
	var secondaryFile CWLSecondaryFileSchema

	patterni, ok := secondaryFileMap["pattern"]
	if !ok {
		return nil, errors.New("")
	}
	pattern, err := getCWLExpressionString(patterni)
	if err != nil {
		return nil, err
	}
	secondaryFile.Pattern = pattern

	requiredi, ok := secondaryFileMap["required"]
	if ok {
		required, err := getCWLExpressionBool(requiredi)
		if err != nil {
			return nil, err
		}
		secondaryFile.Required = required

	}

	return &secondaryFile, nil
}

func getCWLSecondaryFileSchemas(secondaryFilesi interface{}) ([]CWLSecondaryFileSchema, error) {
	secondaryFiles := make([]CWLSecondaryFileSchema, 0)

	switch val := secondaryFilesi.(type) {
	case string:
		secondaryFiles = append(secondaryFiles, getCWLSecondaryFileSchemaString(val))
		return secondaryFiles, nil
	case []string:
		for _, file := range val {
			secondaryFiles = append(secondaryFiles, getCWLSecondaryFileSchemaString(file))
		}
	case map[string]interface{}:
		break
	case []map[string]interface{}:
		break
	default:
		return nil, errors.New("")
	}
	return nil, nil
}

func getCWLLoadListingEnum(listingEnum interface{}) (LoadListingEnum, error) {
	_, ok := listingEnum.(string)
	if !ok {
		return nil, errors.New("")
	}

	switch listingEnum {
	case "no_listing":
		return NoListing{}, nil
	case "shallow_listing":
		return ShallowListing{}, nil
	case "deep_listing":
		return DeepListing{}, nil
	default:
		return nil, fmt.Errorf("Expected no_listing | shallow_listing | deep_listing but got %s", listingEnum)
	}
}

func getCLIRecordField(rfield map[string]interface{}) (*CommandlineInputRecordField, error) {
	var recordField CommandlineInputRecordField

	namei, ok := rfield["name"]
	if !ok {
		return nil, errors.New("<name> was expected")
	}

	name, ok := namei.(string)
	if !ok {
		return nil, errors.New("<name> was not of type string")
	}
	recordField.Name = name

	typeVali, ok := rfield["type"]
	if !ok {
		return nil, errors.New("<type> was expected")
	}

	typeVal, err := getCLIRecordType(typeVali)
	if err != nil {
		return nil, err
	}
	recordField.Type = typeVal

	doc, ok := rfield["doc"]
	if ok {
		doc, ok := doc.([]string)
		if !ok {
			return nil, errors.New("<doc> expects type string")
		}
		recordField.Doc = doc
	}

	label, ok := rfield["label"]
	if ok {
		label, ok := label.(string)
		if !ok {
			return nil, errors.New("<label> expects string")
		}
		recordField.Label = &label
	}

	secondaryFiles, ok := rfield["secondaryFiles"]
	if ok {
		secondaryFiles, err := getCWLSecondaryFileSchemas(secondaryFiles)
		if err != nil {
			return nil, errors.New("")
		}
		recordField.SecondaryFiles = secondaryFiles
	}

	streamable, ok := rfield["streamable"]
	if ok {
		streamable, ok := streamable.(bool)
		if !ok {
			return nil, errors.New("streamable must bool")
		}
		recordField.Streamable = &streamable
	}

	format, ok := rfield["format"]
	if ok {
		format, err := getCWLFormat(format)
		if err != nil {
			return nil, err
		}
		recordField.Format = format
	}

	loadContents, ok := rfield["loadContents"]
	if ok {
		loadContents, ok := loadContents.(bool)
		if !ok {
			return nil, errors.New("")
		}
		recordField.LoadContents = &loadContents
	}

	loadListing, ok := rfield["loadListing"]
	if ok {
		loadListing, err := getCWLLoadListingEnum(loadListing)
		if err != nil {
			return nil, err
		}
		recordField.LoadListing = loadListing
	}

	inputBinding, ok := rfield["inputBinding"]
	if ok {
		inputBinding, err := getInputBinding(inputBinding)
		if err != nil {
			return nil, err
		}
		recordField.InputBinding = inputBinding
	}

	return &recordField, nil
}

/*
func getCLIRecordSchema(rschema map[string]interface{}) (*CommandlineInputRecordSchema, error) {
	typeVal, ok := rschema["type"]
	if !ok {
		return nil, errors.New("<type> expected in definition of RecordSchema")
	}

	if typeVal != "record" {
		return nil, errors.New("<type> must be \"record\"")
	}

	fields, ok := rschema["fields"]
	if ok {
		switch fieldsVal := fields.(type) {
		case []map[string]interface{}:
			break
		case map[string]interface{}:
			for name, val := range fieldsVal {
				f, err := getCLIScalarType(val, &name)
				if err != nil {
					return nil, err
				}
			}
		default:
			return nil, errors.New("<fields> in defintion of RecordSchema must be []CommandInputRecordField or map[name]<type>|CommandInputRecordField")
		}
	}

	label, ok := rschema["label"]
	if ok {
	}

	doc, ok := rschema["doc"]
	if ok {
	}

	name, ok := rschema["name"]
	if ok {
	}

	inputBinding, ok := rschema["inputBinding"]
	if ok {
	}
	return nil, nil
} */

func getCLIEnumSchema(eschema map[string]interface{}) (*CommandlineInputEnumSchema, error) {
	var enumSchema CommandlineInputEnumSchema

	symbols, ok := eschema["symbols"]
	if !ok {
		return nil, errors.New("<symbols> expected")
	}
	symbolsArray, ok := symbols.([]string)
	if !ok {
		return nil, errors.New("<symbols> was not of type []string")
	}
	enumSchema.Symbols = symbolsArray

	typeVal, ok := eschema["type"]
	if !ok {
		return nil, errors.New("<type> was expected")
	}

	if typeVal != "enum" {
		return nil, errors.New("<type> was expected to be \"enum\"")
	}

	return &enumSchema, nil
}

func getCLIScalarType(typeValue interface{}, defaultName *string) (CommandlineInputParameterType, error) {
	switch val := typeValue.(type) {
	case string:
		switch val {
		case "int":
			return CWLString{}, nil
		case "null":
			return CWLNull{}, nil
		case "boolean":
			return CWLBool{}, nil
		case "long":
			return CWLLong{}, nil
		case "float":
			return CWLFloat{}, nil
		case "double":
			return CWLDouble{}, nil
		case "string":
			return CWLString{}, nil
		case "File":
			return CWLFile{}, nil
		case "Directory":
			return CWLDirectory{}, nil
		default:
			return nil, fmt.Errorf("<%s> is not a valid type", val)
		}
	case map[string]interface{}:
		typeString, ok := typeValue.(string)
		if !ok {
			return nil, fmt.Errorf("Expected <type> to be string but got %T", typeValue)
		}
		switch typeString {
		case "array":
			break
		case "record":
			break
		case "enum":
			break
		default:
			return nil, fmt.Errorf("\"array\"|\"record\"|\"enum\" was expected but got %s", typeString)
		}
	default:
		return nil, errors.New("")
	}

	return nil, nil
}

func getCLIType(typeValue interface{}) ([]CommandlineInputParameterType, error) {
	inputTypes := make([]CommandlineInputParameterType, 0)
	tys, ok := typeValue.([]interface{})
	if ok {
		for _, ity := range tys {
			ty, err := getCLIScalarType(ity, nil)
			if err != nil {
				return nil, err
			}
			inputTypes = append(inputTypes, ty)
		}
		return inputTypes, nil
	} else {
		ty, err := getCLIScalarType(typeValue, nil)
		if err != nil {
			return nil, err
		}
		inputTypes = append(inputTypes, ty)
		return inputTypes, nil
	}
}

func getCWLExpressionInner(str string) *string {
	if len(str) < 3 {
		return nil
	}

	if str[0] != '$' {
		return nil
	}

	if str[1] == '[' && str[len(str)-1] == ']' {
		return &str
	}

	if str[1] == '{' && str[len(str)-1] == '}' {
		return &str
	}

	return nil
}

func getCWLExpressionBool(exprOrBool interface{}) (CWLExpressionBool, error) {
	switch val := exprOrBool.(type) {
	case bool:
		return Bool(val), nil
	case string:
		inner := getCWLExpressionInner(val)
		if inner != nil {
			return &CWLExpression{Expression: *inner}, nil
		}
		return nil, errors.New("Invalid expression")
	default:
		return nil, errors.New("")
	}
}

func getCWLExpressionString(exprOrString interface{}) (CWLExpressionString, error) {
	str, ok := exprOrString.(string)
	if !ok {
		return nil, fmt.Errorf("Expected <string | expression> but got %T", exprOrString)
	}
	val := getCWLExpressionInner(str)
	if val != nil {
		return CWLExpression{Expression: *val}, nil
	}
	return String(str), nil
}

func getInputBinding(inputBinding interface{}) (*CommandlineBinding, error) {
	var commandlineBinding CommandlineBinding

	inputBindingMap, ok := inputBinding.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("<inputBinding> expected map[string]interface{} but got %T", inputBinding)
	}

	loadContents, ok := inputBindingMap["loadContents"]
	if ok {
		loadContents, ok := loadContents.(bool)
		if !ok {
			return nil, fmt.Errorf("<loadContents> expected bool but got %T", loadContents)
		}
		commandlineBinding.LoadContents = &loadContents
	}

	position, ok := inputBindingMap["position"]
	if ok {
		position, ok := position.(int)
		if !ok {
			return nil, fmt.Errorf("<position> expected int but got %T", position)
		}
		commandlineBinding.Position = &position
	}

	prefix, ok := inputBindingMap["prefix"]
	if ok {
		prefix, ok := prefix.(string)
		if !ok {
			return nil, fmt.Errorf("<prefix> expected string but got %T", prefix)
		}
		commandlineBinding.Prefix = &prefix
	}

	separate, ok := inputBindingMap["separate"]
	if ok {
		separate, ok := separate.(bool)
		if !ok {
			return nil, fmt.Errorf("<seperate> expected bool but got %T", separate)
		}
		commandlineBinding.Seperate = &separate
	}

	itemSeparator, ok := inputBindingMap["itemSeparator"]
	if ok {
		itemSeparator, ok := itemSeparator.(string)
		if !ok {
			return nil, fmt.Errorf("<itemSeparator> expected string but got %T", itemSeparator)
		}
		commandlineBinding.ItemSeperator = &itemSeparator
	}

	valueFrom, ok := inputBindingMap["valueFrom"]
	if ok {
		valueFrom, err := getCWLExpressionString(valueFrom)
		if err != nil {
			return nil, err
		}
		commandlineBinding.ValueFrom = valueFrom
	}

	shellQuote, ok := inputBindingMap["shellQuote"]
	if ok {
		shellQuote, ok := shellQuote.(bool)
		if !ok {
			return nil, fmt.Errorf("<shellQuote> expected bool but got %T", separate)
		}
		commandlineBinding.ShellQuote = &shellQuote
	}

	return &commandlineBinding, nil
}

func getCWLFormat(format interface{}) (CWLFormat, error) {
	return nil, nil
}

func getDoc(doc interface{}) ([]string, error) {
	return nil, nil
}

func getCommandlineInput(input map[string]interface{}, defaultId *string) (*CommandlineInputParameter, error) {
	var parsedInput CommandlineInputParameter

	id, ok := input["id"]
	if !ok {
		if defaultId == nil {
			return nil, errors.New("<id> was not present")
		}
		id = *defaultId
	}

	idString, ok := id.(string)
	if !ok {
		return nil, errors.New("<id> was not of type \"string\"")
	}
	parsedInput.Id = &idString

	label, ok := input["label"]
	if ok {
		labelString, ok := label.(string)
		if !ok {
			return nil, errors.New("<label> was not of type \"string\"")
		}
		parsedInput.Label = &labelString
	}

	secondaryFiles, ok := input["secondaryFiles"]
	if ok {
		secondaryFiles, err := getCWLSecondaryFileSchemas(secondaryFiles)
		if err != nil {
			return nil, err
		}
		parsedInput.SecondaryFiles = secondaryFiles
	}

	streamable, ok := input["streamable"]
	if ok {
		streamable, ok := streamable.(bool)
		if !ok {
			return nil, errors.New("Expected bool")
		}
		parsedInput.Streamable = &streamable

	}

	doc, ok := input["doc"]
	if ok {
		doc, err := getDoc(doc)
		if err != nil {
			return nil, err
		}
		parsedInput.Doc = doc
	}

	format, ok := input["format"]
	if ok {
		format, err := getCWLFormat(format)
		if err != nil {
			return nil, err
		}
		parsedInput.Format = format
	}

	inputBinding, ok := input["inputBinding"]
	if ok {
		inputBinding, err := getInputBinding(inputBinding)
		if err != nil {
			return nil, err
		}
		parsedInput.InputBinding = inputBinding
	}

	defaultValue, ok := input["default"]
	if ok {
		parsedInput.Default = &defaultValue
	}

	typeValue, ok := input["type"]
	if ok {
		typeValue, err := getCLIType(typeValue)
		if err != nil {
			return nil, err
		}
		parsedInput.Type = typeValue
	}
	return &parsedInput, nil
}

func fillCommandlineInputsFromArray(commandlineTool *CommandlineTool, arrayInput []map[string]interface{}) error {
	inputs := make([]CommandlineInputParameter, 0)

	for _, input := range arrayInput {
		input, err := getCommandlineInput(input, nil)
		if err != nil {
			return err
		}
		inputs = append(inputs, *input)
	}
	commandlineTool.Inputs = inputs
	return nil
}

func fillCommandlineInputsFromMap(commandlineTool *CommandlineTool, mappedInputs map[string]map[string]interface{}) error {
	inputs := make([]CommandlineInputParameter, 0)

	for key, input := range mappedInputs {
		input, err := getCommandlineInput(input, &key)
		if err != nil {
			return err
		}
		if *input.Id != key {
			return errors.New("Ambiguous <id> was given")
		}
		inputs = append(inputs, *input)
	}
	commandlineTool.Inputs = inputs
	return nil
}

func FillCommandlineInputs(commandlineTool *CommandlineTool, dynYaml map[string]interface{}) error {
	inputs, ok := dynYaml["inputs"]
	if !ok {
		return errors.New("Could not find \"inputs\"")
	}

	if inputsArray, ok := inputs.([]map[string]interface{}); ok {
		return fillCommandlineInputsFromArray(commandlineTool, inputsArray)
	}

	if inputsMap, ok := inputs.(map[string]interface{}); ok {
		newInputsMap := make(map[string]map[string]interface{})

		for key, value := range inputsMap {
			childMap, ok := value.(map[string]interface{})
			if !ok {
				return errors.New("input entry must be of type map[string]interface{}")
			}
			newInputsMap[key] = childMap
		}

		return fillCommandlineInputsFromMap(commandlineTool, newInputsMap)
	}

	return errors.New("key <inputs> has value not of type \"array[any]\" or \"map[string]any\"")
}

func getCommandlineOutputParamaterType(typeValue interface{}) ([]CommandlineOutputParameterType, error) {
	return nil, nil
}

func getOutputBinding(outputBinding interface{}) (*CommandlineOutputBinding, error) {
	var commandlineOutputBinding CommandlineOutputBinding

	return &commandlineOutputBinding, nil
}

func getCommandlineOutput(output map[string]interface{}, defaultId *string) (*CommandlineOutputParameter, error) {
	var parsedOutput CommandlineOutputParameter

	typeValue, ok := output["type"]
	if ok {
		typeValue, err := getCommandlineOutputParamaterType(typeValue)
		if err != nil {
			return nil, err
		}
		parsedOutput.Type = typeValue
	}

	label, ok := output["label"]
	if ok {
		label, ok := label.(string)
		if !ok {
			return nil, errors.New("<label> was not of type string")
		}
		parsedOutput.Label = &label
	}

	secondaryFiles, ok := output["secondaryFiles"]
	if ok {
		secondaryFiles, err := getCWLSecondaryFileSchemas(secondaryFiles)
		if err != nil {
			return nil, err
		}
		parsedOutput.SecondaryFiles = secondaryFiles
	}

	streamable, ok := output["streamable"]
	if ok {
		streamable, ok := streamable.(bool)
		if !ok {
			return nil, errors.New("<streamable> was not of type bool")
		}
		parsedOutput.Streamable = &streamable
	}

	doc, ok := output["doc"]
	if ok {
		doc, err := getDoc(doc)
		if err != nil {
			return nil, err
		}
		parsedOutput.Doc = doc

	}

	id, ok := output["id"]
	if !ok {
		if defaultId == nil {
			return nil, errors.New("<id> was not present")
		}
		id = *defaultId
	}

	idString, ok := id.(string)
	if !ok {
		return nil, errors.New("<id> was not of type \"string\"")
	}
	parsedOutput.Id = &idString

	format, ok := output["format"]
	if ok {
		format, err := getCWLFormat(format)
		if err != nil {
			return nil, err
		}
		parsedOutput.Format = format
	}

	outputBinding, ok := output["outputBinding"]
	if ok {
		outputBinding, err := getOutputBinding(outputBinding)
		if err != nil {
			return nil, err
		}
		parsedOutput.OutputBinding = outputBinding
	}
	return &parsedOutput, nil
}

func fillCommandlineOutputsFromArray(commandlineTool *CommandlineTool, outputsArray []map[string]interface{}) error {
	outputs := make([]CommandlineOutputParameter, 0)

	for _, output := range outputsArray {
		output, err := getCommandlineOutput(output, nil)
		if err != nil {
			return err
		}
		outputs = append(outputs, *output)
	}
	return nil
}

func fillCommandlineOutputsFromMap(commandlineTool *CommandlineTool, outputsMap map[string]map[string]interface{}) error {
	outputs := make([]CommandlineOutputParameter, 0)

	for key, output := range outputsMap {
		output, err := getCommandlineOutput(output, &key)
		if err != nil {
			return err
		}
		outputs = append(outputs, *output)
	}
	return nil
}

func FillCommandlineOutputs(commandlineTool *CommandlineTool, dynYaml map[string]interface{}) error {
	outputs, ok := dynYaml["outputs"]
	if !ok {
		return nil
	}
	if outputsArray, ok := outputs.([]map[string]interface{}); ok {
		return fillCommandlineOutputsFromArray(commandlineTool, outputsArray)
	}
	if outputsMap, ok := outputs.(map[string]map[string]interface{}); ok {
		return fillCommandlineInputsFromMap(commandlineTool, outputsMap)
	}
	return nil
}

func FillCommandlineToolClass(commandlineTool *CommandlineTool, dynYaml map[string]interface{}) error {
	class, ok := dynYaml["class"]
	if !ok {
		return errors.New("Could not find \"class\"")
	}

	var classString string
	if classString, ok = class.(string); !ok {
		return errors.New("key <class> has value not of type \"string\"")
	}

	if classString != CommandlineToolClass {
		return fmt.Errorf("key <class> has value %s where %s was expected", classString, CommandlineToolClass)
	}

	commandlineTool.Class = classString

	return nil
}

/*
 * ptr is a pointer to a pointer of a string
 * This is because a pointer to a string may be nil
 * in this case it obviously does not make any sense to
 * deref and assign a value to that.
 * Instead we take the pointer to the pointer (ptr)
 * and update the pointer to the string (*ptr = &valueString)
 */
func fillString(ptr **string, dynYaml map[string]interface{}, key string) error {
	value, ok := dynYaml[key]
	if !ok {
		// this should be null but its clearer to set it explicitly
		ptr = nil
		// id does not have to specified
		return nil
	}

	valueString, ok := value.(string)
	if !ok {
		return fmt.Errorf("Expected type string was not received")
	}

	*ptr = &valueString

	return nil
}

func FillCommandlineId(commandlineTool *CommandlineTool, dynYaml map[string]interface{}) error {
	return fillString(&commandlineTool.Id, dynYaml, "id")
}

func FillCommandlineLabel(commandlineTool *CommandlineTool, dynYaml map[string]interface{}) error {
	return fillString(&commandlineTool.Label, dynYaml, "label")
}

func getDockerRequirement(requirement map[string]interface{}) (*DockerRequirement, error) {
	d := DockerRequirement{}

	classi, ok := requirement["class"]
	if !ok {
		return nil, errors.New("<class> required in DockerRequirement definition")
	}
	if classi != "DockerRequirement" {
		return nil, fmt.Errorf("Expected \"DockerRequirement\" got %v", classi)
	}
	var err error

	err = fillString(&d.DockerPull, requirement, "dockerPull")
	if err != nil {
		return nil, err
	}

	err = fillString(&d.DockerLoad, requirement, "dockerLoad")
	if err != nil {
		return nil, err
	}

	err = fillString(&d.DockerFile, requirement, "dockerFile")
	if err != nil {
		return nil, err
	}

	err = fillString(&d.DockerImport, requirement, "dockerImport")
	if err != nil {
		return nil, err
	}

	err = fillString(&d.DockerImageId, requirement, "dockerImageId")
	if err != nil {
		return nil, err
	}

	err = fillString(&d.DockerOutputDirectory, requirement, "dockerOutputDirectory")

	return &d, nil
}

func FillCommandlineRequirements(commandlineTool *CommandlineTool, dynYaml map[string]interface{}) error {
	requirementsi, ok := dynYaml["requirements"]
	if !ok {
		return nil
	}

	requirements, ok := requirementsi.([]interface{})
	if !ok {
		return errors.New("<requirements> expected []map[string]interface{}")
	}

	processedRequirements := make([]CWLRequirements, 0)

	for _, requirementi := range requirements {
		requirementMap, ok := requirementi.(map[string]interface{})
		if !ok {
			return errors.New("<requirements>.element must be map[string]interface{}")
		}

		classi, ok := requirementMap["class"]
		if !ok {
			return errors.New("<requirement.class> must exist")
		}

		switch classi {
		case "DockerRequirement":
			d, err := getDockerRequirement(requirementMap)
			if err != nil {
				return nil
			}
			processedRequirements = append(processedRequirements, d)
		case "InlineJavascriptRequirement":
			break
		default:
			return fmt.Errorf("%v is unexpected", classi)
		}
	}

	if len(processedRequirements) > 0 {
		commandlineTool.Requirements = processedRequirements
	}

	return nil
}

func FillCommandlineBaseCommand(commandlineTool *CommandlineTool, dynYaml map[string]interface{}) error {
	baseCommand, ok := dynYaml["baseCommand"]
	if !ok {
		return nil
	}
	switch baseCommand := baseCommand.(type) {
	case string:
		commandlineTool.BaseCommand = []string{baseCommand}
	case []string:
		commandlineTool.BaseCommand = baseCommand
	default:
		return fmt.Errorf("Expected type array[string] or string but got %T", baseCommand)
	}
	return nil
}

func FillCommandlineTool(dynYaml map[string]interface{}) (*CommandlineTool, error) {
	commandlineTool := &CommandlineTool{}
	var err error

	err = FillCommandlineInputs(commandlineTool, dynYaml)
	if err != nil {
		return nil, err
	}

	err = FillCommandlineOutputs(commandlineTool, dynYaml)
	if err != nil {
		return nil, err
	}

	err = FillCommandlineToolClass(commandlineTool, dynYaml)
	if err != nil {
		return nil, err
	}

	err = FillCommandlineId(commandlineTool, dynYaml)
	if err != nil {
		return nil, err
	}

	err = FillCommandlineLabel(commandlineTool, dynYaml)
	if err != nil {
		return nil, err
	}

	err = FillCommandlineRequirements(commandlineTool, dynYaml)
	if err != nil {
		return nil, err
	}

	err = FillCommandlineBaseCommand(commandlineTool, dynYaml)
	if err != nil {
		return nil, err
	}

	return commandlineTool, nil
}
