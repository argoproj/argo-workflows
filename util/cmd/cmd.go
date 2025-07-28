// Package cmd provides functionally common to various argo CLIs
package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// NewVersionCmd returns a new `version` command to be used as a sub-command to root
func NewVersionCmd(cliName string) *cobra.Command {
	var short bool
	versionCmd := cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			version := argo.GetVersion()
			PrintVersion(cliName, version, short)
			return nil
		},
	}
	versionCmd.Flags().BoolVar(&short, "short", false, "print just the version number")
	return &versionCmd
}

func PrintVersion(cliName string, version wfv1.Version, short bool) {
	fmt.Printf("%s: %s\n", cliName, version.Version)
	if short {
		return
	}
	fmt.Printf("  BuildDate: %s\n", version.BuildDate)
	fmt.Printf("  GitCommit: %s\n", version.GitCommit)
	fmt.Printf("  GitTreeState: %s\n", version.GitTreeState)
	if version.GitTag != "" {
		fmt.Printf("  GitTag: %s\n", version.GitTag)
	}
	fmt.Printf("  GoVersion: %s\n", version.GoVersion)
	fmt.Printf("  Compiler: %s\n", version.Compiler)
	fmt.Printf("  Platform: %s\n", version.Platform)
}

// PrintVersionMismatchWarning detects if there's a mismatch between the client and server versions and prints a warning if so
func PrintVersionMismatchWarning(clientVersion wfv1.Version, serverVersion string) {
	if serverVersion != "" && clientVersion.GitTag != "" && serverVersion != clientVersion.Version {
		log.Warnf("CLI version (%s) does not match server version (%s). This can lead to unexpected behavior.", clientVersion.Version, serverVersion)
	}
}

// MustIsDir returns whether or not the given filePath is a directory. Exits if path does not exist
func MustIsDir(filePath string) bool {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Fatal(err)
	}
	return fileInfo.IsDir()
}

// IsURL returns whether a string is a URL
func IsURL(u string) bool {
	var parsedURL *url.URL
	var err error

	parsedURL, err = url.ParseRequestURI(u)
	if err == nil {
		if parsedURL != nil && parsedURL.Host != "" {
			return true
		}
	}
	return false
}

// ParseLabels turns a string representation of a label set into a map[string]string
func ParseLabels(labelSpec interface{}) (map[string]string, error) {
	labelString, isString := labelSpec.(string)
	if !isString {
		return nil, fmt.Errorf("expected string, found %v", labelSpec)
	}
	if len(labelString) == 0 {
		return nil, fmt.Errorf("no label spec passed")
	}
	labels := map[string]string{}
	labelSpecs := strings.Split(labelString, ",")
	for ix := range labelSpecs {
		labelSpec := strings.Split(labelSpecs[ix], "=")
		if len(labelSpec) != 2 {
			return nil, fmt.Errorf("unexpected label spec: %s", labelSpecs[ix])
		}
		if len(labelSpec[0]) == 0 {
			return nil, fmt.Errorf("unexpected empty label key")
		}
		labels[labelSpec[0]] = labelSpec[1]
	}
	return labels, nil
}

// SetLogFormatter sets a log formatter for logrus
func SetLogFormatter(logFormat string) {
	timestampFormat := "2006-01-02T15:04:05.000Z"
	switch strings.ToLower(logFormat) {
	case "json":
		log.SetFormatter(&log.JSONFormatter{TimestampFormat: timestampFormat})
	case "text":
		log.SetFormatter(&log.TextFormatter{
			TimestampFormat: timestampFormat,
			FullTimestamp:   true,
		})
	default:
		log.Fatalf("Unknown log format '%s'", logFormat)
	}
}
