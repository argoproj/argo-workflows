package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
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
