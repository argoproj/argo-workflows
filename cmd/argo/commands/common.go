package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// Global variables
var (
	jobStatusIconMap         map[wfv1.NodePhase]string
	nodeTypeIconMap          map[wfv1.NodeType]string
	workflowConditionIconMap map[wfv1.ConditionType]string
	noColor                  bool
)

func init() {
	cobra.OnInitialize(initializeSession)
}

// ANSI escape codes
const (
	escape    = "\x1b"
	noFormat  = 0
	Bold      = 1
	FgBlack   = 30
	FgRed     = 31
	FgGreen   = 32
	FgYellow  = 33
	FgBlue    = 34
	FgMagenta = 35
	FgCyan    = 36
	FgWhite   = 37
	FgDefault = 39
)

func initializeSession() {
	jobStatusIconMap = map[wfv1.NodePhase]string{
		wfv1.NodePending:   ansiFormat("◷", FgYellow),
		wfv1.NodeRunning:   ansiFormat("●", FgCyan),
		wfv1.NodeSucceeded: ansiFormat("✔", FgGreen),
		wfv1.NodeSkipped:   ansiFormat("○", FgDefault),
		wfv1.NodeFailed:    ansiFormat("✖", FgRed),
		wfv1.NodeError:     ansiFormat("⚠", FgRed),
	}
	nodeTypeIconMap = map[wfv1.NodeType]string{
		wfv1.NodeTypeSuspend: ansiFormat("ǁ", FgCyan),
	}
	workflowConditionIconMap = map[wfv1.ConditionType]string{
		wfv1.ConditionTypeMetricsError: ansiFormat("✖", FgRed),
		wfv1.ConditionTypeSpecWarning:  ansiFormat("⚠", FgYellow),
	}
}

func ansiColorCode(s string) int {
	i := 0
	for _, c := range s {
		i += int(c)
	}
	colors := []int{FgRed, FgGreen, FgYellow, FgBlue, FgMagenta, FgCyan, FgWhite}
	return colors[i%len(colors)]
}

// ansiFormat wraps ANSI escape codes to a string to format the string to a desired color.
// NOTE: we still apply formatting even if there is no color formatting desired.
// The purpose of doing this is because when we apply ANSI color escape sequences to our
// output, this confuses the tabwriter library which miscalculates widths of columns and
// misaligns columns. By always applying a ANSI escape sequence (even when we don't want
// color, it provides more consistent string lengths so that tabwriter can calculate
// widths correctly.
func ansiFormat(s string, codes ...int) string {
	if noColor || os.Getenv("TERM") == "dumb" || len(codes) == 0 {
		return s
	}
	codeStrs := make([]string, len(codes))
	for i, code := range codes {
		codeStrs[i] = strconv.Itoa(code)
	}
	sequence := strings.Join(codeStrs, ";")
	return fmt.Sprintf("%s[%sm%s%s[%dm", escape, sequence, s, escape, noFormat)
}
