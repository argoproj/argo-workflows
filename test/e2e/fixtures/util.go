package fixtures

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func errorln(args ...any) {
	_, _ = fmt.Fprint(os.Stderr, args...)
}

func Exec(ctx context.Context, name string, stdin string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if stdin != "" {
		data, err := os.ReadFile(stdin)
		if err != nil {
			return "", err
		}
		cmd.Stdin = bytes.NewReader(data)
	}
	cmd.Env = os.Environ()
	if dir := os.Getenv("E2E_CLI_GOCOVERDIR"); dir != "" && filepath.Base(name) == "argo" {
		// The CLI is built with -cover (make COVER=true). Give each call a
		// directory of its own: every process rewrites the shared meta-data
		// file at startup, and a race on it prints an error into the output
		// the tests parse.
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		coverDir, err := os.MkdirTemp(dir, "argo-")
		if err != nil {
			return "", err
		}
		cmd.Env = append(cmd.Env, "GOCOVERDIR="+coverDir)
	}
	_, _ = fmt.Println(cmd.String())
	output, err := runWithTimeout(cmd)
	// Command completed before timeout. Print output and error if it exists.
	if err != nil {
		errorln(err)
	}
	for s := range strings.SplitSeq(output, "\n") {
		_, _ = fmt.Println(s)
	}
	return output, err
}

func runWithTimeout(cmd *exec.Cmd) (string, error) {
	// https://medium.com/@vCabbage/go-timeout-commands-with-os-exec-commandcontext-ba0c861ed738
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Start()
	if err != nil {
		return "", err
	}
	done := make(chan error)
	go func() { done <- cmd.Wait() }()
	timeout := time.After(60 * time.Second)
	select {
	case <-timeout:
		_ = cmd.Process.Kill()
		return buf.String(), fmt.Errorf("timeout")
	case err := <-done:
		return buf.String(), err
	}
}

// LoadObject is used to load yaml to runtime.Object
func LoadObject(text string) (runtime.Object, error) {
	var yaml string
	if after, ok := strings.CutPrefix(text, "@"); ok {
		file := after
		f, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			return nil, err
		}
		yaml = string(f)
	} else {
		yaml = text
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(yaml), nil, nil)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
