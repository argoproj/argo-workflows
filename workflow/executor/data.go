package executor

import (
	"context"
	"encoding/json"
	"fmt"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"k8s.io/utils/pointer"
	"regexp"
	"strings"
)

func (we *WorkflowExecutor) Data(ctx context.Context) error {
	dataTemplate := we.Template.Data
	if dataTemplate == nil {
		return nil
	}

	// Once we allow input parameters to the data template, we'll load them here
	var data interface{}
	if len(we.Template.Inputs.Parameters) == 1 {
		data = &we.Template.Inputs.Parameters[0].Value
	}

	var err error
	for _, step := range dataTemplate {
		switch {
		case step.WithArtifactPaths != nil:
			data, err = we.processWithArtifactPaths(ctx, step.WithArtifactPaths)
		case step.Filter != nil:
			data, err = we.processFilter(data, step.Filter)
		case step.Aggregator != nil:
			data, err = we.processAggregator(data, step.Aggregator)
		}
		if err != nil {
			return fmt.Errorf("error processing data step '%s': %w", step.Name, err)
		}
	}

	return we.processOutput(ctx, data)
}

func (we *WorkflowExecutor) processWithArtifactPaths(ctx context.Context, artifacts *wfv1.WithArtifactPaths) ([]string, error) {
	driverArt, err := we.newDriverArt(&artifacts.Artifact)
	if err != nil {
		return nil, err
	}
	artDriver, err := we.InitDriver(ctx, driverArt)
	if err != nil {
		return nil, err
	}

	var files []string
	files, err = artDriver.ListObjects(&artifacts.Artifact)
	if err != nil {
		return nil, err
	}

	return files, nil
}

func (we *WorkflowExecutor) processFilter(data interface{}, filter *wfv1.Filter) (interface{}, error) {
	switch data.(type) {
	case []string:
		return processFilterSlice(data.([]string), filter)
	case [][]string:
		var out [][]string
		for i, slice := range data.([][]string) {
			filtered, err := processFilterSlice(slice, filter)
			if err != nil {
				return nil, fmt.Errorf("cannot filter index '%d' of data: %w", i, err)
			}
			out = append(out, filtered)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported data type for filtering: %T", data)
	}
}

func processFilterSlice(files []string, filter *wfv1.Filter) ([]string, error) {
	switch fil := filter; {
	case fil.Directory != nil:
		// If recursive is set to false, remove all files that contain a directory
		if fil.Directory.Recursive != nil && !*fil.Directory.Recursive {
			return inPlaceFilter(func(file string) bool {
				return !strings.Contains(file, "/")
			}, files), nil
		}

		if fil.Directory.Regex != "" {
			re, err := regexp.Compile(fil.Directory.Regex)
			if err != nil {
				return nil, fmt.Errorf("regex '%s' is not valid: %w", fil.Directory.Regex, err)
			}
			return inPlaceFilter(func(file string) bool {
				return re.MatchString(file)
			}, files), nil
		}
	}
	return files, nil
}

func (we *WorkflowExecutor) processAggregator(data interface{}, aggregator *wfv1.Aggregator) ([][]string, error) {
	var files []string
	var ok bool
	if files, ok = data.([]string); !ok {
		return nil, fmt.Errorf("intput is not []string")
	}

	var aggFiles [][]string
	switch {
	case aggregator.Batch != 0:
		// Starts at -1 because we increment before first use
		filesSeen := -1
		aggFiles = groupBy(func(file string) string {
			filesSeen++
			return fmt.Sprint(filesSeen / aggregator.Batch)
		}, files)
	case aggregator.Regex != "":
		re, err := regexp.Compile(aggregator.Regex)
		if err != nil {
			return nil, fmt.Errorf("regex '%s' is not valid: %w", aggregator.Regex, err)
		}
		aggFiles = groupBy(func(file string) string {
			match := re.FindStringSubmatch(file)
			if len(match) == 1 {
				return match[0]
			}
			return match[1]
		}, files)
	}

	return aggFiles, nil
}

func (we *WorkflowExecutor) processOutput(ctx context.Context, data interface{}) error {
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}
	we.Template.Outputs.Result = pointer.StringPtr(string(out))
	err = we.AnnotateOutputs(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}

func inPlaceFilter(filter func(file string) bool, files []string) []string {
	keptFiles := 0
	for _, file := range files {
		if filter(file) {
			files[keptFiles] = file
			keptFiles++
		}
	}
	out := files[:keptFiles]
	return out
}

func groupBy(grouper func(file string) string, files []string) [][]string {
	var groups [][]string
	groupIds := make(map[string]int)
	for _, file := range files {
		group := grouper(file)
		id, ok := groupIds[group]
		if !ok {
			groupIds[group] = len(groups)
			id = len(groups)
			groups = append(groups, []string{})
		}
		// IDEA gives a warning here, but we guarantee that groups[id] is not nil above
		groups[id] = append(groups[id], file)
	}
	return groups
}
