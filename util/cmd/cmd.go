// Package cmd provides functionally common to various argo CLIs

package cmd

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/user"

	"github.com/argoproj/argo"
	"github.com/spf13/cobra"
)

// NewVersionCmd returns a new `version` command to be used as a sub-command to root
func NewVersionCmd(cliName string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: fmt.Sprintf("Print version information"),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s version %s\n", cliName, argo.FullVersion)
		},
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

// MustHomeDir returns the home directory of the user
func MustHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}

// IsURL returns whether or not a string is a URL
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
