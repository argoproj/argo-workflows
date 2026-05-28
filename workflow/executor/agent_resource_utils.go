package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/itchyny/gojq"
	"github.com/tidwall/gjson"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	kubectlcmd "k8s.io/kubectl/pkg/cmd"
	kubectlutil "k8s.io/kubectl/pkg/cmd/util"

	argoerrors "github.com/argoproj/argo-workflows/v4/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// gjsonLabels is an implementation of labels.Labels that backs the label
// matching by gjson lookups against an arbitrary JSON document — used to
// evaluate success/failure conditions against a resource manifest.
type gjsonLabels struct {
	json []byte
}

// Has returns whether the provided label exists.
func (g gjsonLabels) Has(label string) bool {
	return gjson.GetBytes(g.json, label).Exists()
}

// Get returns the value for the provided label.
func (g gjsonLabels) Get(label string) string {
	return gjson.GetBytes(g.json, label).String()
}

// Lookup returns the value for the provided label and whether it exists.
func (g gjsonLabels) Lookup(label string) (string, bool) {
	result := gjson.GetBytes(g.json, label)
	return result.String(), result.Exists()
}

func matchConditions(ctx context.Context, jsonBytes []byte, successReqs labels.Requirements, failReqs labels.Requirements) (bool, error) {
	logger := logging.RequireLoggerFromContext(ctx)
	ls := gjsonLabels{json: jsonBytes}
	for _, req := range failReqs {
		failed := req.Matches(ls)
		msg := fmt.Sprintf("failure condition '%s' evaluated %v", req, failed)
		logger.Info(ctx, msg)
		if failed {
			// We return false here to not retry when failure conditions met.
			return false, argoerrors.Errorf(argoerrors.CodeBadRequest, "%s", msg)
		}
	}
	numMatched := 0
	for _, req := range successReqs {
		matched := req.Matches(ls)
		logger.WithFields(logging.Fields{"condition": req, "matched": matched}).Info(ctx, "success condition evaluated")
		if matched {
			numMatched++
		}
	}
	logger.WithFields(logging.Fields{"numMatched": numMatched, "total": len(successReqs)}).Info(ctx, "success conditions matched")
	if numMatched >= len(successReqs) {
		return false, nil
	}

	return true, argoerrors.Errorf(argoerrors.CodeNotFound, "Neither success condition nor the failure condition has been matched. Retrying...")
}

func jqFilter(ctx context.Context, input []byte, filter string) (string, error) {
	var v any
	if err := json.Unmarshal(input, &v); err != nil {
		return "", err
	}
	q, err := gojq.Parse(filter)
	if err != nil {
		return "", err
	}
	iter := q.RunWithContext(ctx, v)
	var buf strings.Builder
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return "", err
		}
		if s, ok := v.(string); ok {
			buf.WriteString(s)
		} else {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			buf.Write(b)
		}
		buf.WriteString("\n")
	}
	return strings.TrimSpace(buf.String()), nil
}

// runKubectlMu serializes runKubectl invocations. The implementation below
// mutates process-global state (os.Args, kubectlutil.BehaviorOnFatal) which
// the agent's parallel task workers would otherwise race on, corrupting each
// other's manifest paths and exit-code closures — observed in production as
// concurrently-created resources receiving each other's monitored-resource
// node-ID labels (so handleDone could not match events back to their tasks).
var runKubectlMu sync.Mutex

// runKubectl is a package-level var so tests can swap it for a stub.
var runKubectl = func(ctx context.Context, args ...string) ([]byte, error) {
	logging.RequireLoggerFromContext(ctx).Info(ctx, strings.Join(args, " "))

	runKubectlMu.Lock()
	defer runKubectlMu.Unlock()

	osArgs := append([]string{}, os.Args...)
	os.Args = args
	defer func() {
		os.Args = osArgs
	}()

	var fatalErr error
	// catch `os.Exit(1)` from kubectl
	kubectlutil.BehaviorOnFatal(func(msg string, code int) {
		fatalErr = argoerrors.New(fmt.Sprint(code), msg)
	})

	var buf bytes.Buffer
	if err := kubectlcmd.NewKubectlCommand(kubectlcmd.KubectlOptions{
		Arguments: args,
		// TODO(vadasambar): use `DefaultConfigFlags` variable from upstream
		// as value for `ConfigFlags` once https://github.com/kubernetes/kubernetes/pull/120024 is merged
		ConfigFlags: genericclioptions.NewConfigFlags(true).
			WithDeprecatedPasswordFlag().
			WithDiscoveryBurst(300).
			WithDiscoveryQPS(50.0),
		IOStreams: genericclioptions.IOStreams{Out: &buf, ErrOut: os.Stderr},
	}).Execute(); err != nil {
		return nil, err
	}
	if fatalErr != nil {
		return nil, fatalErr
	}
	return buf.Bytes(), nil
}
