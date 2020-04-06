package fixtures

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	executil "github.com/argoproj/pkg/exec"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func runCli(args []string) (string, error) {
	runArgs := append([]string{"-n", Namespace}, args...)
	cmd := exec.Command("../../dist/argo", runArgs...)
	cmd.Env = os.Environ()
	output, err := executil.RunCommandExt(cmd, executil.CmdOpts{Timeout: 30 * time.Second})
	level := log.DebugLevel
	if err != nil {
		level = log.ErrorLevel
	}
	log.WithFields(log.Fields{"args": args, "output": output, "err": err}).Log(level, "Run CLI")
	return output, err
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
