package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// groupSize is how many pod nodes hang off each synthetic StepGroup.
const groupSize = 50

type exampleSpec struct {
	path string
	wf   wfv1.Workflow
}

func loadSpecs(ctx context.Context, dir string) ([]exampleSpec, error) {
	var specs []exampleSpec
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || (!strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml")) {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		wfs, err := common.SplitWorkflowYAMLFile(ctx, body, false)
		if err != nil {
			return nil // not a Workflow manifest, skip
		}
		for _, wf := range wfs {
			if len(wf.Spec.Templates) > 0 {
				specs = append(specs, exampleSpec{path: path, wf: wf})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].path < specs[j].path })
	if len(specs) == 0 {
		return nil, fmt.Errorf("no workflow specs found under %s", dir)
	}
	return specs, nil
}

const suffixChars = "abcdefghijklmnopqrstuvwxyz0123456789"

func randSuffix(rng *rand.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = suffixChars[rng.Intn(len(suffixChars))]
	}
	return string(b)
}

func nodeID(wfName, nodeName string) string {
	if wfName == nodeName {
		return wfName
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(nodeName))
	return fmt.Sprintf("%s-%d", wfName, h.Sum32())
}

var failureMessages = []string{
	"OOMKilled (exit code 137)",
	"Error (exit code 1)",
	"pod deleted during operation",
	`failed to pull image "argoproj/argosay:v2": rpc error: code = Unknown`,
}

var paramNames = []string{"message", "image", "replicas", "config", "input-path"}

// synthesizeNodes builds a plausible node-status map for the given spec,
// growing it until it holds at least target nodes. All randomness comes from
// rng so identical seeds give identical corpora.
func synthesizeNodes(spec *exampleSpec, target int, rng *rand.Rand) wfv1.Nodes {
	wfName := spec.wf.Name
	if wfName == "" {
		wfName = strings.TrimSuffix(spec.wf.GenerateName, "-")
	}
	wfName = fmt.Sprintf("%s-%s", wfName, randSuffix(rng, 5))
	scope := "local/" + wfName
	base := time.Unix(1767225600, 0).UTC() // fixed epoch for reproducibility

	templates := spec.wf.Spec.Templates

	// Per-workflow value pool: values repeat across nodes, as parameter values
	// do in real fan-outs.
	values := make([]string, 8)
	for i := range values {
		values[i] = fmt.Sprintf(`{"bucket":"argo-artifacts","key":"runs/%s/item-%d.json","retries":%d}`, wfName, i, rng.Intn(4))
	}

	nodes := wfv1.Nodes{}
	rootName := wfName
	root := wfv1.NodeStatus{
		ID:            nodeID(wfName, rootName),
		Name:          rootName,
		DisplayName:   rootName,
		Type:          wfv1.NodeTypeSteps,
		TemplateName:  templates[0].Name,
		TemplateScope: scope,
		Phase:         wfv1.NodeRunning,
		StartedAt:     metav1.Time{Time: base},
		Progress:      wfv1.Progress("0/1"),
	}

	i := 0
	var groupID string
	for len(nodes) < target-1 { // -1 leaves room for the root added at the end
		if i%groupSize == 0 {
			groupName := fmt.Sprintf("%s[%d]", wfName, i/groupSize)
			groupID = nodeID(wfName, groupName)
			nodes[groupID] = wfv1.NodeStatus{
				ID:            groupID,
				Name:          groupName,
				DisplayName:   fmt.Sprintf("[%d]", i/groupSize),
				Type:          wfv1.NodeTypeStepGroup,
				TemplateScope: scope,
				Phase:         wfv1.NodeSucceeded,
				BoundaryID:    root.ID,
				StartedAt:     metav1.Time{Time: base.Add(time.Duration(i) * time.Second)},
				FinishedAt:    metav1.Time{Time: base.Add(time.Duration(i+groupSize*60) * time.Second)},
				Progress:      wfv1.Progress("1/1"),
			}
			root.Children = append(root.Children, groupID)
		}

		tmpl := templates[i%len(templates)]
		podName := fmt.Sprintf("%s[%d].%s(%d:item-%d)", wfName, i/groupSize, tmpl.Name, i, i%len(values))
		parentID := groupID

		// ~10% of pods sit under a retry node, like real retryStrategy fan-outs.
		retried := rng.Float64() < 0.1
		if retried {
			retryID := nodeID(wfName, podName)
			child := podName + "(0)"
			nodes[retryID] = wfv1.NodeStatus{
				ID:            retryID,
				Name:          podName,
				DisplayName:   fmt.Sprintf("%s(%d:item-%d)", tmpl.Name, i, i%len(values)),
				Type:          wfv1.NodeTypeRetry,
				TemplateName:  tmpl.Name,
				TemplateScope: scope,
				Phase:         wfv1.NodeSucceeded,
				BoundaryID:    root.ID,
				StartedAt:     metav1.Time{Time: base.Add(time.Duration(i) * time.Second)},
				FinishedAt:    metav1.Time{Time: base.Add(time.Duration(i+120) * time.Second)},
				Progress:      wfv1.Progress("1/1"),
				Children:      []string{nodeID(wfName, child)},
			}
			appendChild(nodes, parentID, retryID)
			parentID = retryID
			podName = child
		}

		pod := wfv1.NodeStatus{
			ID:            nodeID(wfName, podName),
			Name:          podName,
			DisplayName:   fmt.Sprintf("%s(%d:item-%d)", tmpl.Name, i, i%len(values)),
			Type:          wfv1.NodeTypePod,
			TemplateName:  tmpl.Name,
			TemplateScope: scope,
			BoundaryID:    root.ID,
			StartedAt:     metav1.Time{Time: base.Add(time.Duration(i) * time.Second)},
			HostNodeName:  fmt.Sprintf("ip-10-0-%d-%d.ec2.internal", rng.Intn(4), rng.Intn(256)),
			PodIP:         fmt.Sprintf("10.244.%d.%d", rng.Intn(8), rng.Intn(256)),
			Inputs: &wfv1.Inputs{
				Parameters: []wfv1.Parameter{
					{Name: paramNames[i%len(paramNames)], Value: wfv1.AnyStringPtr(values[i%len(values)])},
					{Name: paramNames[(i+1)%len(paramNames)], Value: wfv1.AnyStringPtr(values[rng.Intn(len(values))])},
				},
			},
		}
		if retried {
			pod.NodeFlag = &wfv1.NodeFlag{Retried: true}
		}

		switch r := rng.Float64(); {
		case r < 0.90:
			pod.Phase = wfv1.NodeSucceeded
			pod.FinishedAt = metav1.Time{Time: base.Add(time.Duration(i+30+rng.Intn(570)) * time.Second)}
			pod.Progress = wfv1.Progress("1/1")
			pod.ResourcesDuration = wfv1.ResourcesDuration{
				corev1.ResourceCPU:    wfv1.ResourceDuration(int64(rng.Intn(600))),
				corev1.ResourceMemory: wfv1.ResourceDuration(int64(rng.Intn(600))),
			}
			exitCode := "0"
			pod.Outputs = &wfv1.Outputs{
				ExitCode: &exitCode,
				Parameters: []wfv1.Parameter{
					{Name: "result-path", Value: wfv1.AnyStringPtr(values[i%len(values)])},
				},
			}
			if rng.Float64() < 0.3 {
				result := fmt.Sprintf(`{"status":"ok","items":[%s{"code":0}],"checksum":"%08x"}`,
					strings.Repeat(`{"code":0,"msg":"processed"},`, 1+rng.Intn(6)), rng.Uint32())
				pod.Outputs.Result = &result
			}
		case r < 0.95:
			pod.Phase = wfv1.NodeRunning
			pod.Progress = wfv1.Progress("0/1")
		default:
			pod.Phase = wfv1.NodeFailed
			pod.Message = failureMessages[rng.Intn(len(failureMessages))]
			pod.FinishedAt = metav1.Time{Time: base.Add(time.Duration(i+30+rng.Intn(570)) * time.Second)}
			pod.Progress = wfv1.Progress("0/1")
			exitCode := "1"
			pod.Outputs = &wfv1.Outputs{ExitCode: &exitCode}
		}

		nodes[pod.ID] = pod
		if !retried {
			appendChild(nodes, parentID, pod.ID)
		}
		i++
	}

	nodes[root.ID] = root
	return nodes
}

func appendChild(nodes wfv1.Nodes, parentID, childID string) {
	parent := nodes[parentID]
	parent.Children = append(parent.Children, childID)
	nodes[parentID] = parent
}
