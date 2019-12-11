package e2e

import (
	"io/ioutil"
	"strings"
)

type Given struct {
	suite        *E2ESuite
	workflowName string
	file         string
}

func (g *Given) Workflow(name, text string) *Given {

	g.workflowName = name
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
	return g
}

func (g *Given) When() *When {
	return &When{given: g}
}
