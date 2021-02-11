package data

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"regexp"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func ProcessInputParameters(parameters []wfv1.Parameter) (interface{}, error) {
	var data interface{}
	// Currently we only allow one input parameter, but it's easy to see why more than one can be useful: merging and
	// transforming different inputs. This will be added.
	raw := parameters[0].Value.String()
	logrus.Infof("SIMON: %s", raw)
	err := json.Unmarshal([]byte(raw), &data)
	if err != nil {
		return nil, fmt.Errorf("unable to parse input parameter: %w", err)
	}

	// This is hacky and not a final solution
	switch data.(type) {
	case []interface{}:
		list := data.([]interface{})
		if len(list) > 0 {
			switch list[0].(type) {
			case string:
				var out []string
				for _, element := range data.([]interface{}) {
					out = append(out, element.(string))
				}
				return out, nil
			}
		}
	}

	return data, nil
}

func ProcessTransformation(data interface{}, transformation *wfv1.Transformation) (interface{}, error) {
	var err error
	for i, step := range *transformation {
		switch {
		case step.Filter != nil:
			data, err = processFilter(data, step.Filter)
		case step.Map != nil:
			data, err = processMap(data, step.Map)
		case step.Group != nil:
			data, err = processGroup(data, step.Group)
		}
		if err != nil {
			return nil, fmt.Errorf("error processing data step %d: %w", i, err)
		}
	}

	return data, nil
}

func processMap(data interface{}, mapTransform *wfv1.MapTransform) (interface{}, error) {
	switch data.(type) {
	case []string:
		return processMapSlice(data.([]string), mapTransform)
	case [][]string:
		var out [][]string
		for i, slice := range data.([][]string) {
			filtered, err := processMapSlice(slice, mapTransform)
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

func processMapSlice(files []string, mapTransform *wfv1.MapTransform) ([]string, error) {
	if mapTransform.Replace != nil {
		return stringMap(func(file string) string {
			return strings.ReplaceAll(file, mapTransform.Replace.Old, mapTransform.Replace.New)
		}, files), nil
	}
	return files, nil
}

func processFilter(data interface{}, filter *wfv1.Filter) (interface{}, error) {
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
	if filter.Regex != "" {
		re, err := regexp.Compile(filter.Regex)
		if err != nil {
			return nil, fmt.Errorf("regex '%s' is not valid: %w", filter.Regex, err)
		}

		return stringFilter(func(file string) bool {
			return re.MatchString(file)
		}, files), nil
	}
	return files, nil
}

func processGroup(data interface{}, group *wfv1.Group) ([][]string, error) {
	var files []string
	var ok bool
	if files, ok = data.([]string); !ok {
		return nil, fmt.Errorf("intput is not []string")
	}

	var aggFiles [][]string
	switch {
	case group.Batch != 0:
		// Starts at -1 because we increment before first use
		filesSeen := -1
		aggFiles = groupBy(func(file string) string {
			filesSeen++
			return fmt.Sprint(filesSeen / group.Batch)
		}, files)
	case group.Regex != "":
		re, err := regexp.Compile(group.Regex)
		if err != nil {
			return nil, fmt.Errorf("regex '%s' is not valid: %w", group.Regex, err)
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

func stringFilter(filter func(file string) bool, files []string) []string {
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

func stringMap(mapFunc func(file string) string, files []string) []string {
	var out []string
	for _, file := range files {
		out = append(out, mapFunc(file))
	}
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
