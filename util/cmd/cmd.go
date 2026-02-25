// Package cmd provides functionally common to various argo CLIs
package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v4"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
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
func PrintVersionMismatchWarning(ctx context.Context, clientVersion wfv1.Version, serverVersion string) {
	log := logging.RequireLoggerFromContext(ctx)
	if serverVersion != "" && clientVersion.GitTag != "" && serverVersion != clientVersion.Version {
		log.WithFields(logging.Fields{"clientVersion": clientVersion.Version, "serverVersion": serverVersion}).Warn(ctx, "CLI version does not match server version. This can lead to unexpected behavior.")
	}
}

// MustIsDir returns whether or not the given filePath is a directory. Exits if path does not exist
func MustIsDir(ctx context.Context, filePath string) bool {
	log := logging.RequireLoggerFromContext(ctx)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.WithError(err).WithFatal().Error(ctx, "Failed to check if file is a directory")
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

// Ensures we have a logger at the specified level
func CmdContextWithLogger(cmd *cobra.Command, logLevel, logType string) (context.Context, logging.Logger, error) {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	level, err := logging.ParseLevel(logLevel)
	if err != nil {
		return nil, nil, err
	}
	format, err := logging.TypeFromString(logType)
	if err != nil {
		return nil, nil, err
	}
	logger := logging.NewSlogLogger(level, format)
	ctx = logging.WithLogger(ctx, logger)

	// Configure logrus for argoproj/pkg which uses it internally
	SetLogrusLevel(level)
	SetLogrusFormatter(format)

	cmd.SetContext(ctx)
	return ctx, logger, nil
}
