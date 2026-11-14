package parameterization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
	"github.com/argoproj/argo-workflows/v4/workflow/artifacts"
)

const artifactChecksumContract = p.Group + "/artifact-checksums"
const maxRetryArtifactBytes int64 = 64 << 20

var nativeArtifactReference = regexp.MustCompile(`^\{\{(?:tasks|steps)\.[a-zA-Z0-9_-]+\.outputs\.artifacts\.[a-zA-Z0-9_-]+\}\}$`)

// Keep artifact sources in native Argo. This subset only reuses leaf-produced,
// unarchived files with a non-sensitive SHA-256 output parameter declared by the
// author. It neither invents outputs nor changes retained node status.
func validateRetryArtifactContract(t wfv1.Template, call wfv1.DAGTask) error {
	if (t.DAG != nil || len(t.Steps) != 0) && (len(t.Inputs.Artifacts) != 0 || len(t.Outputs.Artifacts) != 0 || len(call.Arguments.Artifacts) != 0) {
		return fmt.Errorf("composite artifact forwarding needs a separate reuse proof")
	}
	if len(t.Inputs.Artifacts) != len(call.Arguments.Artifacts) {
		return fmt.Errorf("artifact retry requires explicit input arguments")
	}
	for _, input := range t.Inputs.Artifacts {
		if input.Optional || input.HasLocation() || input.From != "" || input.FromExpression != "" || call.Arguments.GetArtifactByName(input.Name) == nil {
			return fmt.Errorf("artifact retry requires explicit unambiguous input sources")
		}
	}
	for _, art := range call.Arguments.Artifacts {
		if art.SubPath != "" || art.FromExpression != "" || art.Optional || (art.Raw == nil && !nativeArtifactReference.MatchString(art.From)) || (art.Raw != nil && art.From != "") {
			return fmt.Errorf("retry artifact arguments require raw data or a direct native output reference")
		}
		if art.Raw == nil && art.HasLocation() {
			return fmt.Errorf("retry artifact source cannot mix a location with an output reference")
		}
		if art.Raw != nil && (art.HTTP != nil || art.S3 != nil || art.Git != nil || art.Artifactory != nil || art.GCS != nil || art.OSS != nil || art.HDFS != nil || art.Azure != nil || art.Plugin != nil) {
			return fmt.Errorf("retry raw artifact has ambiguous location")
		}
	}
	if len(t.Outputs.Artifacts) == 0 {
		return nil
	}
	var checksums map[string]string
	if json.Unmarshal([]byte(t.Metadata.Annotations[artifactChecksumContract]), &checksums) != nil || len(checksums) != len(t.Outputs.Artifacts) {
		return fmt.Errorf("artifact outputs require an exact artifact-checksums contract")
	}
	for _, art := range t.Outputs.Artifacts {
		if art.Optional || art.Archive == nil || art.Archive.None == nil || art.From != "" || art.FromExpression != "" || (art.HTTP == nil || art.S3 != nil || art.Artifactory != nil) {
			return fmt.Errorf("retry supports explicit unarchived HTTP output files")
		}
		parameter := checksums[art.Name]
		if parameter == "" || artifactChecksumParameter(t.Outputs, parameter) == nil {
			return fmt.Errorf("artifact checksum must name a declared native output parameter")
		}
	}
	return nil
}

// Check storage using the same native drivers and caller-scoped Kubernetes
// reads as the native artifact endpoint. Locations are frozen Workflow outputs,
// never new URLs from the retry request. Errors never include URLs or contents.
func (s Store) verifyRetainedArtifacts(ctx context.Context, wf *wfv1.Workflow, plan *PartialRetryPlan) error {
	return verifyRetainedArtifactStreams(ctx, wf, plan, func(ctx context.Context, art *wfv1.Artifact) (io.ReadCloser, error) {
		driver, err := artifacts.NewDriver(ctx, art, retryArtifactResources{s})
		if err != nil {
			return nil, err
		}
		return driver.OpenStream(ctx, art)
	})
}

type artifactStream func(context.Context, *wfv1.Artifact) (io.ReadCloser, error)

func verifyRetainedArtifactStreams(ctx context.Context, wf *wfv1.Workflow, plan *PartialRetryPlan, open artifactStream) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	remaining := maxRetryArtifactBytes
	checked := map[string]bool{}
	count := 0
	for id := range plan.RetainedNodes {
		node := wf.Status.Nodes[id]
		if node.Type != wfv1.NodeTypePod || node.Phase != wfv1.NodeSucceeded {
			continue
		}
		t := wf.GetTemplateByName(node.TemplateName)
		if t != nil && len(t.Outputs.Artifacts) == 0 {
			continue
		}
		if node.Outputs == nil || len(node.Outputs.Artifacts) == 0 || (t != nil && len(node.Outputs.Artifacts) != len(t.Outputs.Artifacts)) {
			return fmt.Errorf("retained artifact output evidence missing")
		}
		if t == nil {
			return fmt.Errorf("retained artifact has no frozen producer")
		}
		var checksums map[string]string
		if json.Unmarshal([]byte(t.Metadata.Annotations[artifactChecksumContract]), &checksums) != nil {
			return fmt.Errorf("retained artifact checksum contract missing")
		}
		for _, art := range node.Outputs.Artifacts {
			// Native archived logs are not declared task output files or reused inputs.
			declared := t.Outputs.GetArtifactByName(art.Name)
			if declared == nil {
				return fmt.Errorf("retained artifact is outside its producer contract")
			}
			parameter := artifactChecksumParameter(*node.Outputs, checksums[art.Name])
			if parameter == nil || parameter.Value == nil {
				return fmt.Errorf("retained artifact checksum result missing")
			}
			expected := parameter.Value.String()
			digest, err := hex.DecodeString(expected)
			if err != nil || len(digest) != sha256.Size || expected != strings.ToLower(expected) {
				return fmt.Errorf("retained artifact checksum result invalid")
			}
			if !art.HasLocation() || (art.HTTP == nil || art.S3 != nil || art.Artifactory != nil) {
				return fmt.Errorf("retained artifact needs an explicit supported storage location")
			}
			key := p.Digest([]any{art.ArtifactLocation, expected})
			if checked[key] {
				continue
			}
			count++
			if count > 32 {
				return fmt.Errorf("artifact reuse exceeds 32 files")
			}
			stream, err := open(ctx, art.DeepCopy())
			if err != nil || stream == nil {
				return fmt.Errorf("retained artifact unavailable or unauthorized")
			}
			hash := sha256.New()
			n, readErr := io.Copy(hash, io.LimitReader(stream, remaining+1))
			closeErr := stream.Close()
			if readErr != nil || closeErr != nil || ctx.Err() != nil {
				return fmt.Errorf("retained artifact verification failed")
			}
			if n > remaining {
				return fmt.Errorf("artifact reuse exceeds 64 MiB verification limit")
			}
			remaining -= n
			if hex.EncodeToString(hash.Sum(nil)) != expected {
				return fmt.Errorf("retained artifact content changed")
			}
			checked[key] = true
		}
	}
	return nil
}

type retryArtifactResources struct{ Store }

func (r retryArtifactResources) GetSecret(ctx context.Context, name, key string) (string, error) {
	obj, err := r.Client.Resource(schema.GroupVersionResource{Version: "v1", Resource: "secrets"}).Namespace(r.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("artifact credential unavailable")
	}
	var secret corev1.Secret
	if runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &secret) != nil {
		return "", fmt.Errorf("artifact credential invalid")
	}
	value, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("artifact credential key unavailable")
	}
	return string(value), nil
}
func (r retryArtifactResources) GetConfigMapKey(ctx context.Context, name, key string) (string, error) {
	obj, err := r.Client.Resource(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}).Namespace(r.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("artifact configuration unavailable")
	}
	var cm corev1.ConfigMap
	if runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cm) != nil {
		return "", fmt.Errorf("artifact configuration invalid")
	}
	value, ok := cm.Data[key]
	if !ok {
		return "", fmt.Errorf("artifact configuration key unavailable")
	}
	return value, nil
}

func artifactChecksumParameter(outputs wfv1.Outputs, name string) *wfv1.Parameter {
	for i := range outputs.Parameters {
		if outputs.Parameters[i].Name == name {
			return &outputs.Parameters[i]
		}
	}
	return nil
}
