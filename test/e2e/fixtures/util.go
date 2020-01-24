package fixtures

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func runCli(diagnostics *Diagnostics, args []string) (string, error) {
	runArgs := append([]string{"-n", Namespace}, args...)
	cmd := exec.Command("../../dist/argo", runArgs...)
	cmd.Env = os.Environ()
	output, err := exec.Command("../../dist/argo", runArgs...).CombinedOutput()
	stringOutput := string(output)
	diagnostics.Log(log.Fields{"args": args, "output": stringOutput, "err": err}, "Run CLI")
	return stringOutput, err
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
