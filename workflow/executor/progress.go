package executor

import (
	"regexp"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

var progressRegexp = regexp.MustCompile("#argo .*progress=([0-9]+/[0-9]+)")
var messageRegexp = regexp.MustCompile(`#argo .*message="([^"]*)"`)

func parseProgressAnnotations(s string) (map[string]string, error) {
	annotations := map[string]string{}
	if matches := messageRegexp.FindStringSubmatch(s); len(matches) == 2 {
		annotations[common.AnnotationKeyNodeMessage] = matches[1]
	}
	if matches := progressRegexp.FindStringSubmatch(s); len(matches) == 2 {
		progress, err := wfv1.ParseProgress(matches[1])
		if err != nil {
			return annotations, err
		} else {
			annotations[common.AnnotationKeyProgress] = string(progress)
		}
	}
	return annotations, nil
}
