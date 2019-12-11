package e2e

import (
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

type Given struct {
	suite        *E2ESuite
	workflowName string
	file         string
}

func (g *Given) Workflow(text string) *Given {

	if strings.HasPrefix(text, "@") {
		g.file = strings.TrimPrefix(text, "@")
	} else {
		content := []byte(text)

		tmpfile, err := ioutil.TempFile("", "argo_test")
		if err != nil {
			panic(err)
		}
		if _, err := tmpfile.Write(content); err != nil {
			panic(err)
		}
		if err := tmpfile.Close(); err != nil {
			panic(err)
		}
		g.file = tmpfile.Name()
	}

	file, err := ioutil.ReadFile(g.file)
	if err != nil {
		panic(err)
	}
	obj := make(map[string]interface{})
	err = yaml.Unmarshal(file, obj)
	if err != nil {
		panic(err)
	}

	g.workflowName = obj["metadata"].(map[interface{}]interface{})["name"].(string)

	return g
}

func (g *Given) When() *When {
	return &When{given: g}
}
