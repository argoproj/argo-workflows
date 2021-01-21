package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/fsnotify/fsnotify"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util/archive"
	"github.com/argoproj/argo/workflow/util/path"
)

// https://github.com/tektoncd/pipeline/tree/master/cmd/entrypoint

var varArgo = func(x string) string {
	return filepath.Join("/var/argo", x)
}

func run(name string, args []string) error {
	exitCode := 130 // special error to indicate problem with emissary itself
	defer func() {
		// write the exit code last, which infos the wait car we are done
		if err := ioutil.WriteFile(varArgo("exitcode"), []byte(strconv.Itoa(exitCode)), 0600); err != nil { // 600 = rw-------
			println(fmt.Sprintf("failed to capture exit code %d: %v", exitCode, err))
		}
	}()
	data, err := ioutil.ReadFile(varArgo("template"))
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}
	template := &wfv1.Template{}

	if err := json.Unmarshal(data, template); err != nil {
		return fmt.Errorf("failed to unmarshal template: %w", err)
	}

	name, err = path.Search(name)
	if err != nil {
		return fmt.Errorf("failed to find name in PATH: %w", err)
	}

	signals := make(chan os.Signal, 1)
	defer close(signals)
	signal.Notify(signals)
	defer signal.Reset()

	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout, err = os.Create(varArgo("stdout"))
	if err != nil {
		return fmt.Errorf("failed to open stdout: %w", err)
	}
	cmd.Stderr, err = os.Create(varArgo("stderr"))
	if err != nil {
		return fmt.Errorf("failed to open stderr: %w", err)
	}
	go func() {
		_ = tail(varArgo("stdout"), os.Stdout)
	}()
	go func() {
		_ = tail(varArgo("stderr"), os.Stderr)
	}()

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create signal watcher: %w", err)
	}
	defer func() { _ = w.Close() }()
	if err := ioutil.WriteFile(varArgo("signal"), nil, 0600); err != nil { // fsnotify can only listen to changes to files
		return fmt.Errorf("failed to create signal file: %w", err)
	}
	if err := w.Add(varArgo("signal")); err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-w.Events:
				data, _ := ioutil.ReadFile(varArgo("signal"))
				s, _ := strconv.Atoi(string(data))
				if s > 0 {
					signals <- syscall.Signal(s)
				}
			case <-w.Errors:
				return
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		for s := range signals {
			if s != syscall.SIGCHLD {
				_ = syscall.Kill(-cmd.Process.Pid, s.(syscall.Signal))
			}
		}
	}()

	err = cmd.Wait()

	if err == nil {
		exitCode = 0
	} else if exitError, ok := err.(*exec.ExitError); ok {
		exitCode = exitError.ExitCode()
	}

	var paths []string
	for _, x := range template.Outputs.Parameters {
		if x.ValueFrom != nil && x.ValueFrom.Path != "" {
			paths = append(paths, x.ValueFrom.Path)
		}
	}
	for _, x := range template.Outputs.Artifacts {
		if x.Path != "" {
			paths = append(paths, x.Path)
		}
	}

	for _, src := range paths {
		if _, err := os.Stat(src); os.IsNotExist(err) { // might be optional, so we ignore
			continue
		}
		dst := filepath.Join(varArgo("outputs"), src+".tgz")
		println(src, "->", dst)
		z := filepath.Dir(dst)
		if err := os.MkdirAll(z, 0700); err != nil { // chmod rwx------
			return fmt.Errorf("failed to create directory %s: %w", z, err)
		}
		err := func() error {
			out, err := os.Create(dst)
			if err != nil {
				return fmt.Errorf("failed to create destination %s: %w", dst, err)
			}
			defer func() { _ = out.Close() }()
			if err = archive.TarGzToWriter(src, gzip.DefaultCompression, out); err != nil {
				return fmt.Errorf("failed to tarball the output %s to %s: %w", src, dst, err)
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}

	return err // this is the error returned from cmd.Wait(), which maybe an exitError
}

func main() {
	err := run(os.Args[1], os.Args[2:])
	if exitError, ok := err.(*exec.ExitError); ok {
		os.Exit(exitError.ExitCode())
	} else if err != nil { // this is probably an error related to the emissary itself, and we use code 129 for those errors
		println(err.Error())
		os.Exit(129)
	}
}
