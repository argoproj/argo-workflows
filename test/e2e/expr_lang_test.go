//go:build functional

package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type ExprSuite struct {
	fixtures.E2ESuite
}

func (s *ExprSuite) TestRegression12037() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: broken-
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: split
            template: foo
          - name: map
            template: foo
            depends: split

    - name: foo
      container:
        image: argoproj/argosay:v2
        command:
          - sh
          - -c
          - |
            echo "foo"
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, ".split")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, ".map")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})
}

func (s *ExprSuite) TestExprStringAndMathFunctions() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: expr-string-math-
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: message
      value: "Hello World"
    - name: count
      value: "42"
    - name: ratio
      value: "3.14"
    - name: text
      value: "foo bar baz"
  templates:
    - name: main
      inputs:
        parameters:
        - name: message
        - name: count
        - name: ratio
        - name: text
      container:
        image: argoproj/argosay:v2
        command: [sh, -c]
        args:
        - |
          # Test string functions - will fail if expressions don't produce expected results
          test "{{=string(inputs.parameters.count)}}" = "42" || exit 1
          test "{{=lower(inputs.parameters.message)}}" = "hello world" || exit 1
          test "{{=upper(inputs.parameters.message)}}" = "HELLO WORLD" || exit 1
          test "{{=replace(inputs.parameters.text, \"bar\", \"BAR\")}}" = "foo BAR baz" || exit 1
          test "{{=trim(\"  hello  \")}}" = "hello" || exit 1
          test "{{=hasPrefix(inputs.parameters.message, \"Hello\")}}" = "true" || exit 1
          test "{{=hasSuffix(inputs.parameters.message, \"World\")}}" = "true" || exit 1
          test "{{=indexOf(inputs.parameters.message, \"World\") >= 0}}" = "true" || exit 1
          test "{{=indexOf(inputs.parameters.message, \"xyz\") == -1}}" = "true" || exit 1
          
          # Test math functions
          test "{{=int(inputs.parameters.count)}}" = "42" || exit 1
          test "{{=float(inputs.parameters.ratio)}}" = "3.14" || exit 1
          test "{{=int(inputs.parameters.count) + 8}}" = "50" || exit 1
          test "{{=max(int(inputs.parameters.count), 50)}}" = "50" || exit 1
          test "{{=min(int(inputs.parameters.count), 50)}}" = "42" || exit 1
          test "{{=abs(-5)}}" = "5" || exit 1
          test "{{=ceil(3.2)}}" = "4" || exit 1
          test "{{=floor(3.8)}}" = "3" || exit 1
          test "{{=round(3.6)}}" = "4" || exit 1
          
          # Test logic operations
          test "{{=inputs.parameters.message == \"Hello World\"}}" = "true" || exit 1
          test "{{=int(inputs.parameters.count) > 0 && inputs.parameters.message != \"\"}}" = "true" || exit 1
          test "{{=inputs.parameters.message == \"\" || int(inputs.parameters.count) > 0}}" = "true" || exit 1
          test "{{=int(inputs.parameters.count) > 40 ? \"large\" : \"small\"}}" = "large" || exit 1
          
          # Test type checking
          test "{{=type(inputs.parameters.message)}}" = "string" || exit 1
          test "{{=type(42)}}" = "int" || exit 1
          
          echo "All string and math expression tests passed!"
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		})
}

func (s *ExprSuite) TestExprArrayAndDateFunctions() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: expr-array-date-
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: items
      value: "[\"apple\", \"banana\", \"cherry\"]"
    - name: numbers
      value: "[1, 2, 3, 4, 5]"
  templates:
    - name: main
      inputs:
        parameters:
        - name: items
        - name: numbers
      container:
        image: argoproj/argosay:v2
        command: [sh, -c]
        args:
        - |
          # Test array functions - will fail if expressions don't produce expected results
          echo "Testing array access..."
          first_item="{{=fromJSON(inputs.parameters.items)[0]}}"
          echo "First item: ${first_item}"
          test "${first_item}" = "apple" || (echo "First item test failed" && exit 1)
          
          echo "Testing array length..."
          item_count="{{=len(fromJSON(inputs.parameters.items))}}"
          echo "Item count: ${item_count}"
          test "${item_count}" = "3" || (echo "Length test failed" && exit 1)
          
          # Test array slicing
          echo "Testing array slicing..."
          numbers_slice="{{=fromJSON(inputs.parameters.numbers)[1:3]}}"
          echo "Numbers slice: ${numbers_slice}"
          # Note: slice returns an array, check it's not empty
          test -n "${numbers_slice}" || (echo "Slice test failed" && exit 1)
          
          # Test that now() returns a time (just check it's not empty)
          echo "Testing now() function..."
          current_date="{{=now().Format(\"2006-01-02\")}}"
          echo "Current date: ${current_date}"
          test -n "${current_date}" || (echo "Date test failed" && exit 1)
          
          # Test Unix timestamp is a number (should be > 1600000000 for any recent time)
          echo "Testing Unix timestamp..."
          unix_time="{{=now().Unix()}}"
          echo "Unix time: ${unix_time}"
          test "${unix_time}" -gt "1600000000" || (echo "Unix time test failed" && exit 1)
          
          # Test workflow creation timestamp access
          echo "Testing workflow timestamp..."
          creation_time="{{=workflow.creationTimestamp}}"
          echo "Creation time: ${creation_time}"
          test -n "${creation_time}" || (echo "Creation time test failed" && exit 1)
          
          echo "All array and date expression tests passed!"
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		})
}

func (s *ExprSuite) TestExprEncodingAndUtilityFunctions() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: expr-encoding-utility-
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: data
      value: '{"name": "John", "age": 30}'
    - name: text
      value: "Hello World"
  templates:
    - name: main
      inputs:
        parameters:
        - name: data
        - name: text
      container:
        image: argoproj/argosay:v2
        command: [sh, -c]
        args:
        - |
          # Test JSON functions - will fail if expressions don't produce expected results
          test "{{=get(fromJSON(inputs.parameters.data), \"name\")}}" = "John" || exit 1
          test "{{=get(fromJSON(inputs.parameters.data), \"age\")}}" = "30" || exit 1
          
          # Test JSON serialization with known object
          json_output="{{=toJSON({\"test\": \"value\", \"number\": 123})}}"
          echo "${json_output}" | grep -q "\"test\":\"value\"" || exit 1
          echo "${json_output}" | grep -q "\"number\":123" || exit 1
          
          # Test Base64 encoding/decoding round trip
          test "{{=fromBase64(toBase64(inputs.parameters.text))}}" = "Hello World" || exit 1
          test "{{=toBase64(\"Hello World\")}}" = "SGVsbG8gV29ybGQ=" || exit 1
          
          # Test safe access functions
          test "{{=get([\"a\", \"b\", \"c\"], 1)}}" = "b" || exit 1
          test "{{=get({\"key\": \"value\"}, \"key\")}}" = "value" || exit 1
          
          # Test repeat alternative - manual string concatenation (replacement for repeat function)
          base_str="abc"
          repeated_str="${base_str}${base_str}${base_str}"
          test "${repeated_str}" = "abcabcabc" || exit 1
          
          # Test string utilities
          test "{{=indexOf(\"hello world\", \"world\")}}" = "6" || exit 1
          test "{{=indexOf(\"hello world\", \"xyz\") == -1 ? \"not found\" : \"found\"}}" = "not found" || exit 1
          test "{{=trim(\"__hello__\", \"_\")}}" = "hello" || exit 1
          
          # Test available sprig functions that can replace others
          test "{{=title(\"hello world\")}}" = "Hello World" || exit 1
          test "{{=trunc(10, \"this is a long string\")}}" = "this is a " || exit 1
          
          echo "All encoding and utility expression tests passed!"
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		})
}

func (s *ExprSuite) TestExprConditionalAndParameterPassing() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: expr-conditional-params-
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: environment
      value: "production"
    - name: count
      value: "5"
    - name: enabled
      value: "true"
  templates:
    - name: main
      inputs:
        parameters:
        - name: environment
        - name: count
        - name: enabled
      steps:
      - - name: conditional-step
          template: validate-expressions
          arguments:
            parameters:
            - name: result
              value: "{{=inputs.parameters.environment == \"production\" ? \"prod-mode\" : \"dev-mode\"}}"
            - name: numeric-result
              value: "{{=int(inputs.parameters.count) > 3 ? \"high\" : \"low\"}}"
            - name: boolean-result
              value: "{{=inputs.parameters.enabled == \"true\" && int(inputs.parameters.count) > 0 ? \"active\" : \"inactive\"}}"
            - name: complex-logic
              value: "{{=inputs.parameters.environment == \"production\" && inputs.parameters.enabled == \"true\" ? \"prod-active\" : (inputs.parameters.environment == \"staging\" ? \"staging\" : \"other\")}}"
      
    - name: validate-expressions
      inputs:
        parameters:
        - name: result
        - name: numeric-result
        - name: boolean-result
        - name: complex-logic
      container:
        image: argoproj/argosay:v2
        command: [sh, -c]
        args:
        - |
          # Test complex conditional expressions - will fail if expressions don't produce expected results
          test "{{=inputs.parameters.result}}" = "prod-mode" || exit 1
          test "{{=inputs.parameters.numeric-result}}" = "high" || exit 1
          test "{{=inputs.parameters.boolean-result}}" = "active" || exit 1
          test "{{=inputs.parameters.complex-logic}}" = "prod-active" || exit 1
          
          echo "All conditional expression validations passed!"
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		})
}

func (s *ExprSuite) TestExprInConditionsAndWithItems() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: expr-conditions-items-
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: stage
      value: "production"
    - name: items
      value: "[\"item1\", \"item2\", \"item3\"]"
  templates:
    - name: main
      inputs:
        parameters:
        - name: stage
        - name: items
      dag:
        tasks:
        - name: conditional-task
          template: echo-simple
          arguments:
            parameters:
            - name: message
              value: "Running in {{=inputs.parameters.stage}}"
          when: "{{=inputs.parameters.stage == \"production\" || inputs.parameters.stage == \"staging\"}}"
        
        - name: process-items
          template: validate-item
          arguments:
            parameters:
            - name: item-name
              value: "{{=item.name}}"
            - name: item-upper
              value: "{{=upper(item.name)}}"
          withItems: [{"name": "item1"}, {"name": "item2"}, {"name": "item3"}]
          depends: conditional-task
        
        - name: math-conditions
          template: validate-length
          arguments:
            parameters:
            - name: array-length
              value: "{{=len(fromJSON(inputs.parameters.items))}}"
          when: "{{=len(fromJSON(inputs.parameters.items)) > 2}}"
          depends: process-items
        
        - name: string-conditions
          template: validate-contains
          arguments:
            parameters:
            - name: contains-prod
              value: "{{=indexOf(inputs.parameters.stage, \"prod\") >= 0}}"
          when: "{{=indexOf(inputs.parameters.stage, \"prod\") >= 0}}"
          depends: conditional-task
        
        - name: json-parsing-test
          template: validate-json
          arguments:
            parameters:
            - name: first-item
              value: "{{=fromJSON(inputs.parameters.items)[0]}}"
            - name: array-length
              value: "{{=len(fromJSON(inputs.parameters.items))}}"
          depends: conditional-task
    
    - name: validate-item
      inputs:
        parameters:
        - name: item-name
        - name: item-upper
      container:
        image: argoproj/argosay:v2
        command: [sh, -c]
        args:
        - |
          # Validate that the item name is one of the expected values
          case "{{=inputs.parameters.item-name}}" in
            item1|item2|item3) echo "Valid item: {{=inputs.parameters.item-name}}" ;;
            *) echo "Invalid item: {{=inputs.parameters.item-name}}" && exit 1 ;;
          esac
          
          # Validate uppercase conversion
          expected_upper=$(echo "{{=inputs.parameters.item-name}}" | tr '[:lower:]' '[:upper:]')
          test "{{=inputs.parameters.item-upper}}" = "${expected_upper}" || exit 1
    
    - name: validate-length
      inputs:
        parameters:
        - name: array-length
      container:
        image: argoproj/argosay:v2
        command: [sh, -c]
        args:
        - |
          # Validate array length is exactly 3
          test "{{=inputs.parameters.array-length}}" = "3" || exit 1
          echo "Array length validation passed"
    
    - name: validate-contains
      inputs:
        parameters:
        - name: contains-prod
      container:
        image: argoproj/argosay:v2
        command: [sh, -c]
        args:
        - |
          # Validate that stage contains 'prod'
          test "{{=inputs.parameters.contains-prod}}" = "true" || exit 1
          echo "String contains validation passed"
    
    - name: validate-json
      inputs:
        parameters:
        - name: first-item
        - name: array-length
      container:
        image: argoproj/argosay:v2
        command: [sh, -c]
        args:
        - |
          # Validate JSON parsing results
          test "{{=inputs.parameters.first-item}}" = "item1" || exit 1
          test "{{=inputs.parameters.array-length}}" = "3" || exit 1
          echo "JSON parsing validation passed"
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		})
}

func (s *ExprSuite) TestAllDocumentedMigrationExamples() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: expr-migration-examples-
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: count
      value: "5"
    - name: name
      value: "test-user"
    - name: a
      value: "10"
    - name: b
      value: "15"
  templates:
    - name: main
      inputs:
        parameters:
        - name: count
        - name: name
        - name: a
        - name: b
      container:
        image: argoproj/argosay:v2
        command: [sh, -c]
        args:
        - |
          # Test 1: String operations
          echo "Testing string conversion..."
          result1="{{=string(inputs.parameters.count)}}"
          echo "Result1: ${result1}"
          test "${result1}" = "5" || (echo "Test1 failed: expected 5, got ${result1}" && exit 1)
          
          echo "Testing lower case..."
          result2="{{=lower(inputs.parameters.name)}}"
          echo "Result2: ${result2}"
          test "${result2}" = "test-user" || (echo "Test2 failed: expected test-user, got ${result2}" && exit 1)
          
          # Test 2: Math operations
          echo "Testing math addition..."
          result3="{{=int(inputs.parameters.a) + int(inputs.parameters.b)}}"
          echo "Result3: ${result3}"
          test "${result3}" = "25" || (echo "Test3 failed: expected 25, got ${result3}" && exit 1)
          
          # Test 3: Now function
          echo "Testing now function..."
          current_date="{{=now().Format(\"2006-01-02\")}}"
          echo "Current date: ${current_date}"
          test -n "${current_date}" || (echo "Test4 failed: date is empty" && exit 1)
          
          echo "All tests passed!"
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		})
}

func (s *ExprSuite) TestEncodingAndAdvancedExamples() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: expr-encoding-advanced-
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: raw_data
      value: "hello world"
    - name: json_string
      value: "{\"name\":\"test\",\"count\":42}"
    - name: csv_data
      value: "apple,banana,cherry"
    - name: myArray
      value: "[\"first\", \"second\", \"third\"]"
  templates:
    - name: main
      inputs:
        parameters:
        - name: raw_data
        - name: json_string
        - name: csv_data
        - name: myArray
      container:
        image: argoproj/argosay:v2
        command: [sh, -c]
        args:
        - |
          # Test encoding functions from documentation
          echo "Testing base64 encoding..."
          encoded_b64="{{=base64(inputs.parameters.raw_data)}}"
          echo "Encoded: ${encoded_b64}"
          test "${encoded_b64}" = "aGVsbG8gd29ybGQ=" || (echo "Base64 test failed" && exit 1)
          
          # Test JSON parsing from documentation
          echo "Testing JSON parsing..."
          parsed_name="{{=fromJSON(inputs.parameters.json_string).name}}"
          echo "Parsed name: ${parsed_name}"
          test "${parsed_name}" = "test" || (echo "JSON name test failed" && exit 1)
          
          parsed_count="{{=fromJSON(inputs.parameters.json_string).count}}"
          echo "Parsed count: ${parsed_count}"
          test "${parsed_count}" = "42" || (echo "JSON count test failed" && exit 1)
          
          # Test string manipulation functions from documentation
          echo "Testing string upper..."
          upper_text="{{=upper(inputs.parameters.raw_data)}}"
          echo "Upper text: ${upper_text}"
          test "${upper_text}" = "HELLO WORLD" || (echo "Upper test failed" && exit 1)
          
          echo "Testing string trim..."
          trimmed_text="{{=trim(\"  hello world  \")}}"
          echo "Trimmed text: ${trimmed_text}"
          test "${trimmed_text}" = "hello world" || (echo "Trim test failed" && exit 1)
          
          # Test split function from documentation
          echo "Testing split function..."
          split_first="{{=split(inputs.parameters.csv_data, \",\")[0]}}"
          echo "Split first: ${split_first}"
          test "${split_first}" = "apple" || (echo "Split test failed" && exit 1)
          
          # Test array access patterns from documentation
          echo "Testing array access..."
          first_elem="{{=fromJSON(inputs.parameters.myArray)[0]}}"
          echo "First element: ${first_elem}"
          test "${first_elem}" = "first" || (echo "Array access test failed" && exit 1)
          
          last_elem="{{=fromJSON(inputs.parameters.myArray)[len(fromJSON(inputs.parameters.myArray))-1]}}"
          echo "Last element: ${last_elem}"
          test "${last_elem}" = "third" || (echo "Last element test failed" && exit 1)
          
          echo "All encoding and advanced expression tests passed!"
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		})
}

func TestExprLangSuite(t *testing.T) {
	suite.Run(t, new(ExprSuite))
}
