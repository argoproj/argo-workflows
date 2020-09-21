package common

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/argoproj/argo/cmd/argo/commands/client"

	"github.com/spf13/cobra"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// Global variables
var (
	JobStatusIconMap         map[wfv1.NodePhase]string
	NodeTypeIconMap          map[wfv1.NodeType]string
	WorkflowConditionIconMap map[wfv1.ConditionType]string
	NoColor                  bool
)
var MissingArgumentsError = fmt.Errorf("missing required argument")

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
	JobStatusIconMap = map[wfv1.NodePhase]string{
		wfv1.NodePending:   ANSIFormat("◷", FgYellow),
		wfv1.NodeRunning:   ANSIFormat("●", FgCyan),
		wfv1.NodeSucceeded: ANSIFormat("✔", FgGreen),
		wfv1.NodeSkipped:   ANSIFormat("○", FgDefault),
		wfv1.NodeFailed:    ANSIFormat("✖", FgRed),
		wfv1.NodeError:     ANSIFormat("⚠", FgRed),
	}
	NodeTypeIconMap = map[wfv1.NodeType]string{
		wfv1.NodeTypeSuspend: ANSIFormat("ǁ", FgCyan),
	}
	WorkflowConditionIconMap = map[wfv1.ConditionType]string{
		wfv1.ConditionTypeMetricsError: ANSIFormat("✖", FgRed),
		wfv1.ConditionTypeSpecWarning:  ANSIFormat("⚠", FgYellow),
	}
}

func ANSIColorCode(s string) int {
	i := 0
	for _, c := range s {
		i += int(c)
	}
	colors := []int{FgRed, FgGreen, FgYellow, FgBlue, FgMagenta, FgCyan, FgWhite}
	return colors[i%len(colors)]
}

// ANSIFormat wraps ANSI escape codes to a string to format the string to a desired color.
// NOTE: we still apply formatting even if there is no color formatting desired.
// The purpose of doing this is because when we apply ANSI color escape sequences to our
// output, this confuses the tabwriter library which miscalculates widths of columns and
// misaligns columns. By always applying a ANSI escape sequence (even when we don't want
// color, it provides more consistent string lengths so that tabwriter can calculate
// widths correctly.
func ANSIFormat(s string, codes ...int) string {
	if NoColor || os.Getenv("TERM") == "dumb" || len(codes) == 0 {
		return s
	}
	codeStrs := make([]string, len(codes))
	for i, code := range codes {
		codeStrs[i] = strconv.Itoa(code)
	}
	sequence := strings.Join(codeStrs, ";")
	return fmt.Sprintf("%s[%sm%s%s[%dm", escape, sequence, s, escape, noFormat)
}

var CreateNewAPIClientFunc = client.NewAPIClient
