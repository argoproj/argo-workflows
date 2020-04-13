package common

import (
	"fmt"
	"regexp"
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
type ByDescendingStringLength []DependsOperand

func (b ByDescendingStringLength) Len() int {
	return len(b)
}

func (b ByDescendingStringLength) Less(i, j int) bool {
	return len(b[i].String()) > len(b[j].String())
}

func (b ByDescendingStringLength) Swap(i, j int) {
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
