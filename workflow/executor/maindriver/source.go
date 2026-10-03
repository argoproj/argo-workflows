package maindriver

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
)

// PodSource yields the one task a pod spec delivers, then is exhausted. In
// the init-less layout, call Next only after the supervisor is READY.
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

var _ TaskSource = &PodSource{}

func (s *PodSource) Next(ctx context.Context) (Task, bool, error) {
	if s.done {
		return Task{}, false, nil
	}
	s.done = true
	if len(s.Command) == 0 {
		return Task{}, false, errors.New("no command to run")
	}
	command, err := s.command(ctx)
	if err != nil {
		return Task{}, false, err
	}
	data, err := s.readTemplate()
	if err != nil {
		return Task{}, false, fmt.Errorf("failed to read template: %w", err)
	}
	tmpl := &wfv1.Template{}
	if err := json.Unmarshal(data, tmpl); err != nil {
		return Task{}, false, fmt.Errorf("failed to unmarshal template: %w", err)
	}
	return Task{
		NodeID:              s.NodeID,
		Template:            tmpl,
		Command:             command,
		Env:                 s.Env,
		IncludeScriptOutput: s.IncludeScriptOutput,
		WorkDir:             filepath.Join(s.VarRunArgo, "ctr", s.ContainerName),
	}, true, nil
}

func (s *PodSource) command(ctx context.Context) ([]string, error) {
	command := append([]string{}, s.Command...)
	// Check if args were offloaded to a file (for large args that exceed exec limit)
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

	// Check for a large args and offload to file if needed
	// This avoids the exec() "argument list too long" error
	// Downstream programs should support @filename for parsing large args
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

// readTemplate returns the serialized template JSON. It prefers
// /var/run/argo/template (legacy: init container wrote it; init-less with
// supervisor: supervisor wrote it), and falls back to the ARGO_TEMPLATE env
// var when the file is absent. This covers the init-less case for templates
// that don't run a supervisor (data, resource-without-logs) — the controller
// sets ARGO_TEMPLATE directly on main in that case.
//
// Offload-sentinel resolution is shared with the legacy init container via
// common.ResolveTemplateEnvValue.
func (s *PodSource) readTemplate() ([]byte, error) {
	filePath := filepath.Join(s.VarRunArgo, "template")
	if data, err := os.ReadFile(filePath); err == nil {
		return data, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	// theory-debt: empty ARGO_TEMPLATE treated as unset (emissary did not).
	if s.TemplateEnv == "" {
		return nil, fmt.Errorf("neither %s nor %s is available", filePath, common.EnvVarTemplate)
	}
	return common.ResolveTemplateEnvValue(s.TemplateEnv, s.OffloadDir)
}
