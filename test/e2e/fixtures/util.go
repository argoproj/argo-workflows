package fixtures

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func runCli(args []string) (string, error) {
	runArgs := append([]string{"-n", Namespace}, args...)
	cmd := exec.Command("../../dist/argo", runArgs...)
	cmd.Env = os.Environ()

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

	timeout := time.After(20 * time.Second)

	select {
	case <-timeout:
		_ = cmd.Process.Kill()
		return "", fmt.Errorf("timout")
	case err := <-done:
		// Command completed before timeout. Print output and error if it exists.
		output := buf.String()
		log.WithFields(log.Fields{"args": args, "output": output, "err": err}).Info("Run CLI")
		return output, err
	}
}

// LoadObject is used to load yaml to runtime.Object
func LoadObject(text string) (runtime.Object, error) {
	var yaml string
	if strings.HasPrefix(text, "@") {
		file := strings.TrimPrefix(text, "@")
		f, err := ioutil.ReadFile(file)
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
