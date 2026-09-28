package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/maindriver"
)

// PodSource yields the single task baked into this pod by the pod spec, then
// is exhausted. Every field is parsed from the environment or argv by the
// caller.
//
// In the init-less layout the supervisor writes the template during Prepare,
// so Next must not be called before the supervisor reports READY.
type PodSource struct {
	// VarRunArgo is the shared argoexec directory, normally /var/run/argo.
	VarRunArgo string
	// TemplateEnv is the value of ARGO_TEMPLATE, used when the template file
	// is absent.
	TemplateEnv string
	// OffloadDir is where an offloaded ARGO_TEMPLATE is resolved from.
	OffloadDir string
	// ArgsFile is the value of ARGO_CONTAINER_ARGS_FILE: a JSON list of
	// extra args appended to Command.
	ArgsFile            string
	NodeID              string
	ContainerName       string
	Command             []string
	Env                 []string
	IncludeScriptOutput bool

	done bool
}

var _ maindriver.TaskSource = &PodSource{}

func (s *PodSource) Next(ctx context.Context) (maindriver.Task, bool, error) {
	if s.done {
		return maindriver.Task{}, false, nil
	}
	s.done = true
	if len(s.Command) == 0 {
		return maindriver.Task{}, false, errors.New("no command to run")
	}
	command, err := s.command(ctx)
	if err != nil {
		return maindriver.Task{}, false, err
	}
	data, err := s.readTemplate()
	if err != nil {
		return maindriver.Task{}, false, fmt.Errorf("failed to read template: %w", err)
	}
	tmpl := &wfv1.Template{}
	if err := json.Unmarshal(data, tmpl); err != nil {
		return maindriver.Task{}, false, fmt.Errorf("failed to unmarshal template: %w", err)
	}
	return maindriver.Task{
		NodeID:              s.NodeID,
		Template:            tmpl,
		Command:             command,
		Env:                 s.Env,
		IncludeScriptOutput: s.IncludeScriptOutput,
		WorkDir:             filepath.Join(s.VarRunArgo, "ctr", s.ContainerName),
	}, true, nil
}

// command appends the args file to Command and offloads any arg too large to
// pass on the command line to a file, passed as @<file>.
func (s *PodSource) command(ctx context.Context) ([]string, error) {
	command := append([]string{}, s.Command...)
	if s.ArgsFile == "" {
		return command, nil
	}
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithField("argsFile", s.ArgsFile).Info(ctx, "Reading container args from file")
	data, err := os.ReadFile(s.ArgsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read container args file %s: %w", s.ArgsFile, err)
	}
	var fileArgs []string
	if err := json.Unmarshal(data, &fileArgs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal container args: %w", err)
	}
	command = append(command, fileArgs...)
	logger.WithField("count", len(fileArgs)).Info(ctx, "Loaded container args from file")

	// Index 0 is the executable; the emissary numbers args from after it.
	for i, arg := range command[1:] {
		if len(arg) > common.MaxEnvVarLen {
			filePath := fmt.Sprintf("/tmp/argo_arg_%d.txt", i)
			if err := os.WriteFile(filePath, []byte(arg), 0o644); err != nil {
				return nil, fmt.Errorf("failed to write large arg %d to file: %w", i, err)
			}
			logger.WithFields(logging.Fields{
				"argIndex": i,
				"size":     len(arg),
				"filePath": filePath,
			}).Info(ctx, "Offloaded large argument to file. Downstream program must support @filename syntax")
			command[i+1] = "@" + filePath
		}
	}
	return command, nil
}

func (s *PodSource) readTemplate() ([]byte, error) {
	filePath := filepath.Join(s.VarRunArgo, "template")
	if data, err := os.ReadFile(filePath); err == nil {
		return data, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	// theory-debt: an empty ARGO_TEMPLATE is treated as unset; the emissary
	// distinguishes them (an empty value then fails to unmarshal instead).
	if s.TemplateEnv == "" {
		return nil, fmt.Errorf("neither %s nor %s is available", filePath, common.EnvVarTemplate)
	}
	return common.ResolveTemplateEnvValue(s.TemplateEnv, s.OffloadDir)
}
