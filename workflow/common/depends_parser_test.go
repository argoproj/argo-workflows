package common

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

type parseDependsLogicTestCase struct {
	depends  string
	expected []DependsOperand
}

func TestParseDependsLogic(t *testing.T) {
	testCases := []parseDependsLogicTestCase{
		{depends: "task1 && task2", expected: []DependsOperand{{TaskName: "task1"}, {TaskName: "task2"}}},
		{depends: "task1.Completed && task2", expected: []DependsOperand{{TaskName: "task1", TaskResult: TaskResultCompleted}, {TaskName: "task2"}}},
		{depends: "(task1.Completed && task2)", expected: []DependsOperand{{TaskName: "task1", TaskResult: TaskResultCompleted}, {TaskName: "task2"}}},
		{depends: "(task1.Completed && !task2)", expected: []DependsOperand{{TaskName: "task1", TaskResult: TaskResultCompleted}, {TaskName: "task2"}}},
		{depends: "(task1.Completed &&!task2)", expected: []DependsOperand{{TaskName: "task1", TaskResult: TaskResultCompleted}, {TaskName: "task2"}}},
		{depends: "(task1.Completed&& !task2)", expected: []DependsOperand{{TaskName: "task1", TaskResult: TaskResultCompleted}, {TaskName: "task2"}}},
		{depends: "(task1.Completed&&!task2)", expected: []DependsOperand{{TaskName: "task1", TaskResult: TaskResultCompleted}, {TaskName: "task2"}}},
		{depends: "(task1.Completed&&!task2)||task3.Failed", expected: []DependsOperand{{TaskName: "task1", TaskResult: TaskResultCompleted}, {TaskName: "task2"}, {TaskName: "task3", TaskResult: TaskResultFailed}}},
		{depends: "(task-1.Completed&&!task-2)", expected: []DependsOperand{{TaskName: "task-1", TaskResult: TaskResultCompleted}, {TaskName: "task-2"}}},
		{depends: "(task_1.Completed&&!task2)", expected: []DependsOperand{{TaskName: "task_1", TaskResult: TaskResultCompleted}, {TaskName: "task2"}}},
		// It's not the responsibility of the parser to error on unknown TaskResults
		{depends: "(task_1.Unknown&&!task2)", expected: []DependsOperand{{TaskName: "task_1", TaskResult: "Unknown"}, {TaskName: "task2"}}},
		{depends: "(task_1.Unknown&&!task2.)", expected: []DependsOperand{{TaskName: "task_1", TaskResult: "Unknown"}, {TaskName: "task2"}}},
		{depends: "(task_1.Unknown&&!task2..)", expected: []DependsOperand{{TaskName: "task_1", TaskResult: "Unknown"}, {TaskName: "task2"}}},
	}

	for _, testCase := range testCases {
		assert.Equal(t, testCase.expected, ParseDependsLogic(testCase.depends))
	}
}

type replaceDependsLogicTestCase struct {
	logic    string
	operands []DependsOperand
	expected string
}

func shuffleDependsOperands(d []DependsOperand) {
	rand.Shuffle(len(d), func(i, j int) {
		d[i], d[j] = d[j], d[i]
	})
}

func TestReplaceDependsLogic(t *testing.T) {
	testCases := []replaceDependsLogicTestCase{
		{
			logic: "task1 && task2",
			operands: []DependsOperand{
				{TaskName: "task1", Satisfied: true},
				{TaskName: "task2", Satisfied: true},
			},
			expected: "true && true",
		},
		{
			logic: "foo.Failed && contains_foo.Failed",
			operands: []DependsOperand{
				{TaskName: "foo", TaskResult: TaskResultFailed, Satisfied: true},
				{TaskName: "contains_foo", TaskResult: TaskResultFailed, Satisfied: true},
			},
			expected: "true && true",
		},
		{
			logic: "(foo.Failed) &&!contains_foo.Failed",
			operands: []DependsOperand{
				{TaskName: "foo", TaskResult: TaskResultFailed, Satisfied: false},
				{TaskName: "contains_foo", TaskResult: TaskResultFailed, Satisfied: true},
			},
			expected: "(false) &&!true",
		},
		{
			logic: "foo.Failed || (!foo.Failed && foo2.Failed)",
			operands: []DependsOperand{
				{TaskName: "foo", TaskResult: TaskResultFailed, Satisfied: true},
				{TaskName: "foo2", TaskResult: TaskResultFailed, Satisfied: false},
			},
			expected: "true || (!true && false)",
		},
		{
			// Here we have an unfortunate task named "true", make sure it doesn't get in the way
			// TODO
			logic: "pass.Failed || true",
			operands: []DependsOperand{
				{TaskName: "pass", TaskResult: TaskResultFailed, Satisfied: true},
				{TaskName: "true", Satisfied: false},
			},
			expected: "true || false",
		},
	}

	for _, testCase := range testCases {
		// Result should be stable
		shuffleDependsOperands(testCase.operands)
		assert.Equal(t, testCase.expected, ReplaceDependsLogic(testCase.logic, testCase.operands))
	}
}
