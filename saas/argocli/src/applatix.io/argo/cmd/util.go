package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"applatix.io/axerror"
)

// checkFatal exits if the supplied error is not nil and prints the error, then exits
func checkFatal(errIf interface{}) {
	if axErr, ok := errIf.(*axerror.AXError); ok {
		if axErr != nil {
			log.Fatalf("%s: %s\n", axErr.Code, axErr.Message)
		}
		return
	}
	if err, ok := errIf.(error); ok {
		if err != nil {
			log.Fatalln(err)
		}
		return
	}
}

// getSetting retrieves the value in the following order of preference (CLI flag, env variable, then default)
func getSetting(flagValue, envVariable, defaultVal string) string {
	if flagValue != "" {
		return flagValue
	}
	envVal, found := os.LookupEnv(envVariable)
	if found {
		return envVal
	}
	return defaultVal
}

// runCmd runs a command and returns its output as a string. exits and logs stderr if command failed
func runCmd(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	outBytes, err := cmd.Output()
	if err != nil {
		exErr := err.(*exec.ExitError)
		cmdStr := path.Base(name) + " " + strings.Join(args, " ")
		log.Fatalf("`%s` failed: %s", cmdStr, exErr.Stderr)
	}
	return string(outBytes)
}

// fileExists returns whether or not a file exists at the given path
func fileExists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}

// runCmdTTY is used for running a command that requires tty
func runCmdTTY(cmdName string, arg ...string) {
	cmd := exec.Command(cmdName, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Failed to start command %s. %s\n", cmdName, err.Error())
		os.Exit(1)
	}
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
)

// ansiFormat wraps ANSI escape codes to a string to format the string to a desired color.
// NOTE: we still apply formatting even if there is no color formatting desired.
// The purpose of doing this is because when we apply ANSI color escape sequences to our
// output, this confuses the tabwriter library which miscalculates widths of columns and
// misaligns columns. By always applying a ANSI escape sequence (even when we don't want
// color, it provides more consistent string lengths so that tabwriter can calculate
// widths correctly.
func ansiFormat(s string, codes ...int) string {
	if globalArgs.noColor || os.Getenv("TERM") == "dumb" || len(codes) == 0 {
		return s
	}
	codeStrs := make([]string, len(codes))
	for i, code := range codes {
		codeStrs[i] = strconv.Itoa(code)
	}
	sequence := strings.Join(codeStrs, ";")
	return fmt.Sprintf("%s[%sm%s%s[%dm", escape, sequence, s, escape, noFormat)
}
