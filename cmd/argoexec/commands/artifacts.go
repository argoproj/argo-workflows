package commands

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(artifactsCmd)
	artifactsCmd.AddCommand(artifactsLoadCmd)
	artifactsCmd.AddCommand(artifactsSaveCmd)
}

var artifactsCmd = &cobra.Command{
	Use:   "artifacts",
	Short: "Artifacts commands",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

var artifactsLoadCmd = &cobra.Command{
	Use:   "load",
	Short: "Load artifacts",
	Run:   loadArtifacts,
}

var artifactsSaveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save artifacts",
	Run:   saveArtifacts,
}

// Open the Kubernetes downward api file and
// read the pod annotation file that contains template and then
// unmarshal the template
func GetTemplateFromPodAnnotations(annotationsPath string, template *wfv1.Template) error {
	// Read the annotation file
	file, err := os.Open(annotationsPath)
	if err != nil {
		fmt.Printf("ERROR opening annotation file from %s\n", annotationsPath)
		return errors.InternalWrapError(err)
	}

	defer file.Close()
	reader := bufio.NewReader(file)

	// Prefix of template property in the annotation file
	prefix := fmt.Sprintf("%s=", common.AnnotationKeyTemplate)

	for {
		// Read line-by-line
		var buffer bytes.Buffer

		var l []byte
		var isPrefix bool
		for {
			l, isPrefix, err = reader.ReadLine()
			buffer.Write(l)

			// If we've reached the end of the line, stop reading.
			if !isPrefix {
				break
			}

			// If we're just at the EOF, break
			if err != nil {
				break
			}
		}

		// The end of the annotation file
		if err == io.EOF {
			break
		}

		line := buffer.String()

		// Read template property
		if strings.HasPrefix(line, prefix) {
			// Trim the prefix
			templateContent := strings.TrimPrefix(line, prefix)

			// This part is a bit tricky in terms of unmarshalling
			// The content in the file will be something like,
			// `"{\"type\":\"container\",\"inputs\":{},\"outputs\":{}}"`
			// which is required to unmarshal twice

			// First unmarshal to a string without escaping characters
			var templateString string
			err = json.Unmarshal([]byte(templateContent), &templateString)
			if err != nil {
				fmt.Printf("Error unmarshalling annotation into template string, %s, %v\n", templateContent, err)
				return errors.InternalWrapError(err)
			}

			// Second unmarshal to a template
			err = json.Unmarshal([]byte(templateString), template)
			if err != nil {
				fmt.Printf("Error unmarshalling annotation into template, %s, %v\n", templateString, err)
				return errors.InternalWrapError(err)
			}
			return nil
		}
	}

	if err != io.EOF {
		return errors.InternalWrapError(err)
	}

	// If reaching here, then no template prefix in the file
	return errors.InternalErrorf("No template property found from annotation file: %s", annotationsPath)
}

func loadArtifacts(cmd *cobra.Command, args []string) {
	wfExecutor := initExecutor()
	// Download input artifacts
	err := wfExecutor.LoadScriptSource()
	if err != nil {
		log.Fatalf("Error loading script: %+v", err)
	}
	err = wfExecutor.LoadArtifacts()
	if err != nil {
		log.Fatalf("Error downloading input artifacts: %+v", err)
	}
	os.Exit(0)
}

func saveArtifacts(cmd *cobra.Command, args []string) {
	wfExecutor := initExecutor()
	err := wfExecutor.SaveArtifacts()
	if err != nil {
		log.Fatalf("Error saving output artifacts, %+v", err)
	}
	err = wfExecutor.CaptureScriptResult()
	if err != nil {
		log.Fatalf("Error capturing script output, %+v", err)
	}
	err = wfExecutor.AnnotateOutputs()
	if err != nil {
		log.Fatalf("Error annotating outputs, %+v", err)
	}
	os.Exit(0)
}
