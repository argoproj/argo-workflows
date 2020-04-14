package common

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const taskResultSeparator = "."

type TaskResult string

const (
	TaskResultSucceeded  TaskResult = "Succeeded"
	TaskResultFailed     TaskResult = "Failed"
	TaskResultSkipped    TaskResult = "Skipped"
	TaskResultCompleted  TaskResult = "Completed"
	TaskResultAny        TaskResult = "Any"
	TaskResultSuccessful TaskResult = "Successful"
)

type DependsOperand struct {
	TaskName   string
	TaskResult TaskResult
	Satisfied  bool
}

func (d *DependsOperand) String() string {
	if d.TaskResult != "" {
		return fmt.Sprintf("%s%s%s", d.TaskName, taskResultSeparator, d.TaskResult)
	}
	return d.TaskName
}

// An interface to sort by descending string length value. This is needed when replacing operands with their boolean
// results
type byDescendingStringLength []DependsOperand

func (b byDescendingStringLength) Len() int {
	return len(b)
}

func (b byDescendingStringLength) Less(i, j int) bool {
	return len(b[i].String()) > len(b[j].String())
}

func (b byDescendingStringLength) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func ParseDependsLogic(depends string) []DependsOperand {
	re := regexp.MustCompile(fmt.Sprintf(`([\w\-]+?\%s[A-Za-z]+)|([\w\-]+)`, taskResultSeparator))
	matches := re.FindAllStringSubmatch(depends, -1)
	var out []DependsOperand
	for _, matchGroup := range matches {
		if matchGroup[1] != "" {
			split := strings.Split(matchGroup[1], taskResultSeparator)
			task, result := split[0], split[1]
			out = append(out, DependsOperand{TaskName: task, TaskResult: TaskResult(result)})
		} else if matchGroup[2] != "" {
			out = append(out, DependsOperand{TaskName: matchGroup[2]})
		}
	}
	return out
}

func ReplaceDependsLogic(logic string, operands []DependsOperand) string {
	// Replace operands with boolean values indicating if they are satisfied. We make string replacements in order of
	// largest string length value to smallest. This is necessary to avoid replacing a subset of a larger string if it
	// happens to be the case that a smaller, valid string is found within it.
	sort.Sort(byDescendingStringLength(operands))
	for _, operand := range operands {
		logic = strings.Replace(logic, operand.String(), fmt.Sprintf("%t", operand.Satisfied), -1)
	}
	return logic
}
