package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func readmeGen() {
	dir, err := ioutil.ReadDir("docs")
	if err != nil {
		panic(err)
	}
	fh, err := os.Create("docs/README.md")
	if err != nil {
		panic(err)
	}
	_, err = fh.WriteString(`# Argo Documentation

### Getting Started

For set-up information and running your first Workflows, please see our [Getting Started](getting-started.md) guide.

### Examples

For detailed examples about what Argo can do, please see our [Argo Workflows: Documentation by Example](../examples/README.md) page.

### Fields

For a full list of all the fields available in for use in Argo, and a link to examples where each is used, please see [Argo Fields](fields.md).

### Features
Some use-case specific documentation is available:

`)
	if err != nil {
		panic(err)
	}
	for _, info := range dir {
		if info.IsDir() {
			continue
		}
		name := info.Name()
		if name == "README.md" {
			continue
		}
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		content, err := ioutil.ReadFile("docs/" + name)
		if err != nil {
			panic(err)
		}
		heading := strings.TrimPrefix(strings.SplitN(string(content), "\n", 2)[0], "# ")
		_, err = fh.WriteString(fmt.Sprintf("* [%s](%s)\n", heading, name))
		if err != nil {
			panic(err)
		}
	}
	err = fh.Close()
	if err != nil {
		panic(err)
	}
}
