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
	Task      string
	Result    TaskResult
	Satisfied bool
}

func (d *DependsOperand) String() string {
	if d.Result != "" {
		return fmt.Sprintf("%s%s%s", d.Task, taskResultSeparator, d.Result)
	}
	return d.Task
}

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
			out = append(out, DependsOperand{Task: task, Result: TaskResult(result)})
		} else if matchGroup[2] != "" {
			out = append(out, DependsOperand{Task: matchGroup[2]})
		}
	}
	return out
}
