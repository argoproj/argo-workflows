package poolpod

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// DirName is the name of the pool directory, Dir, within /var/run/argo.
const DirName = "pool"

// TaskFile is the file in each task directory that holds the task's
// TaskSpec. The worker writes it before it offers the task.
const TaskFile = "task"

// TaskSpec is what the worker tells main about one task: everything main
// needs to run the task's command as a pod's main container would run it.
// What a pool pod's main container has whatever the task (the env from
// valueFrom and envFrom, the image's working directory, the volume mounts)
// is main's own and isn't repeated here.
type TaskSpec struct {
	// NodeID is the task's node ID.
	NodeID string `json:"nodeID"`
	// Template is the template as a pod's executor gets it: its inputs and
	// outputs say what main links into place and stages, and its archive
	// location whether main captures the logs.
	Template *wfv1.Template `json:"template"`
	// Command is the container's command, as written in the template:
	// main expands its $(VAR) references. Empty when the template has
	// none, and then ImageCommand is run.
	Command []string `json:"command,omitempty"`
	// ImageCommand is the image's entrypoint, followed by its cmd when Args
	// is empty, as dispatch resolved it. main runs it as it is when Command
	// is empty: the kubelet doesn't expand an image's entrypoint or cmd.
	ImageCommand []string `json:"imageCommand,omitempty"`
	// Args follow the command: the container's args as written, then, for
	// a script with source, the path of the source staged for this task.
	// main expands their $(VAR) references.
	Args []string `json:"args,omitempty"`
	// Env is the task's own environment, applied in order over main's: the
	// template's plain value entries and the ARGO_* variables that identify
	// the task. Values are as written in the template: main expands their
	// $(VAR) references, each against main's environment and the entries
	// before it, as the kubelet does.
	Env []EnvVar `json:"env,omitempty"`
	// IncludeScriptOutput makes main capture stdout for the script result.
	IncludeScriptOutput bool `json:"includeScriptOutput,omitempty"`
}

// EnvVar is one of a task's environment variables. It has no valueFrom:
// those are resolved by the kubelet when the pool pod starts, so they are
// in main's own environment.
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (s TaskSpec) validate() error {
	switch {
	case s.NodeID == "":
		return errors.New("task has no node ID")
	case s.Template == nil:
		return errors.New("task has no template")
	case len(s.Command) == 0 && len(s.ImageCommand) == 0:
		return errors.New("task has no command to run")
	}
	return nil
}

// WriteSpec writes the task's spec into its directory, replacing it
// atomically. Called by the worker before Offer.
func (t Task) WriteSpec(spec TaskSpec) error {
	if err := spec.validate(); err != nil {
		return err
	}
	data, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	return writeAtomic(t.dir, TaskFile, data)
}

// Spec reads the task's spec. Called by main once it has taken the task.
func (t Task) Spec() (TaskSpec, error) {
	data, err := os.ReadFile(filepath.Join(t.dir, TaskFile))
	if err != nil {
		return TaskSpec{}, err
	}
	var spec TaskSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return TaskSpec{}, fmt.Errorf("parse %s: %w", filepath.Join(t.dir, TaskFile), err)
	}
	if err := spec.validate(); err != nil {
		return TaskSpec{}, fmt.Errorf("%s: %w", filepath.Join(t.dir, TaskFile), err)
	}
	return spec, nil
}

// CheckTaskName fails for a name that can't name a task's directory in the
// pool directory: it must be a single path element, and must not be one of
// the pool directory's own files (`current`, main's liveness lock `main`,
// and the dot-files its atomic writes go through).
func CheckTaskName(name string) error {
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") || name == CurrentFile || name == MainFile {
		return fmt.Errorf("%q cannot name a pool task's directory: it must be a single path element that doesn't start with %q and isn't %q or %q", name, ".", CurrentFile, MainFile)
	}
	return nil
}

// Resolve returns the command line and environment main runs the task
// with, given main's own environment, resolved as the kubelet resolves a
// container's: each of the task's env values is expanded against environ
// and the task's entries before it, then the command and args against the
// whole environment. The environment is environ followed by the task's
// entries, which win over environ's.
//
// One difference: environ includes the image's own env, which the kubelet
// doesn't expand references to.
func (s TaskSpec) Resolve(environ []string) (argv, env []string) {
	vars := make(map[string]string, len(environ)+len(s.Env))
	for _, kv := range environ {
		if name, value, ok := strings.Cut(kv, "="); ok {
			vars[name] = value
		}
	}
	// Never nil: a nil Env makes os/exec inherit this process's
	// environment instead.
	env = append(make([]string, 0, len(environ)+len(s.Env)), environ...)
	for _, e := range s.Env {
		value := expand(e.Value, vars)
		vars[e.Name] = value
		env = append(env, e.Name+"="+value)
	}
	if len(s.Command) > 0 {
		for _, c := range s.Command {
			argv = append(argv, expand(c, vars))
		}
	} else {
		argv = append(argv, s.ImageCommand...)
	}
	for _, a := range s.Args {
		argv = append(argv, expand(a, vars))
	}
	return argv, env
}

// expand replaces each $(VAR) in input with VAR's value in vars, as the
// kubelet does: a reference to an undefined variable is left as written,
// $$ is an escaped $, and any other $ is kept.
func expand(input string, vars map[string]string) string {
	var b strings.Builder
	for i := 0; i < len(input); i++ {
		if input[i] != '$' || i+1 == len(input) {
			b.WriteByte(input[i])
			continue
		}
		switch next := input[i+1]; next {
		case '$':
			b.WriteByte('$')
			i++
		case '(':
			end := strings.IndexByte(input[i+2:], ')')
			if end < 0 {
				// An unterminated reference is kept as written.
				b.WriteString("$(")
				i++
				continue
			}
			name := input[i+2 : i+2+end]
			if value, ok := vars[name]; ok {
				b.WriteString(value)
			} else {
				b.WriteString("$(" + name + ")")
			}
			i += 2 + end
		default:
			b.WriteByte('$')
			b.WriteByte(next)
			i++
		}
	}
	return b.String()
}
