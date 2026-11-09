package common

import (
	"fmt"
	"hash/fnv"
	"io"

	"github.com/spf13/pflag"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	kubectlcreate "k8s.io/kubectl/pkg/cmd/create"
	kubectlutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/yaml"

	argoerrors "github.com/argoproj/argo-workflows/v4/errors"
)

// ReplaceGenerateNameWithDeterministic replaces generateName with a deterministic name
// derived from the node ID, enabling idempotent resource creation on executor pod recovery.
// Uses the same fnv32a hashing pattern as GeneratePodName and Workflow.NodeID.
func ReplaceGenerateNameWithDeterministic(manifest, nodeID string) string {
	obj := unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(manifest), &obj); err != nil {
		return manifest
	}
	if obj.GetName() != "" || obj.GetGenerateName() == "" {
		return manifest
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(nodeID))
	prefix := obj.GetGenerateName()
	// Keep the identity suffix even when the user supplies a long prefix.
	if len(prefix) > 55 {
		prefix = prefix[:55]
	}
	name := fmt.Sprintf("%s%08x", prefix, h.Sum32())
	obj.SetName(name)
	obj.SetGenerateName("")
	bytes, err := yaml.Marshal(obj.Object)
	if err != nil {
		return manifest
	}
	return string(bytes)
}

// ResourceRecoveryGetFlags parses with kubectl's own create flags, then keeps
// client configuration for the get request. Create-only flags such as --validate
// are not accepted by get. Additional input sources cannot be replayed safely:
// only the staged manifest has deterministic identity and ownership annotations.
func ResourceRecoveryGetFlags(flags []string) ([]string, error) {
	if len(flags) == 0 {
		return nil, nil
	}
	config := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	clientFlags := pflag.NewFlagSet("resource-recovery", pflag.ContinueOnError)
	config.AddFlags(clientFlags)
	cmd := kubectlcreate.NewCmdCreate(kubectlutil.NewFactory(config), genericclioptions.IOStreams{
		Out: io.Discard, ErrOut: io.Discard,
	})
	cmd.Flags().AddFlagSet(clientFlags)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.ParseFlags(flags); err != nil {
		return nil, fmt.Errorf("parse resource create flags for recovery: %w", err)
	}
	for _, name := range []string{"filename", "kustomize", "raw", "edit"} {
		if cmd.Flags().Changed(name) {
			return nil, argoerrors.Errorf(argoerrors.CodeBadRequest, "cannot recover resource create with --%s", name)
		}
	}
	if len(cmd.Flags().Args()) != 0 {
		return nil, argoerrors.New(argoerrors.CodeBadRequest, "cannot recover resource create with positional arguments")
	}
	var result []string
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		if clientFlags.Lookup(flag.Name) == nil {
			return
		}
		if value, ok := flag.Value.(pflag.SliceValue); ok {
			for _, item := range value.GetSlice() {
				result = append(result, "--"+flag.Name+"="+item)
			}
			return
		}
		result = append(result, "--"+flag.Name+"="+flag.Value.String())
	})
	return result, nil
}
