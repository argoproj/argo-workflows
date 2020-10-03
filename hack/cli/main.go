package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra/doc"

	"github.com/argoproj/argo/cmd/argo/commands"
)

func main() {
	println("generating docs/cli")
	cmd := commands.NewCommand()
	cmd.DisableAutoGenTag = true
	err := removeContents("docs/cli")
	if err != nil {
		panic(err)
	}
	err = doc.GenMarkdownTree(cmd, "docs/cli")
	if err != nil {
		panic(err)
	}
}

func removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}
