package grifts

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	. "github.com/markbates/grift/grift"
)

var _ = Add("release", func(c *Context) error {
	cmd := exec.Command("git", "tag", "--list")
	if b, err := cmd.CombinedOutput(); err == nil {
		lines := bytes.Split(b, []byte("\n"))
		for _, l := range lines[len(lines)-6:] {
			fmt.Println(string(l))
		}
	}

	r := bufio.NewReader(os.Stdin)
	fmt.Print("Enter version number (vx.x.x): ")
	v, _ := r.ReadString('\n')
	v = strings.TrimSpace(v)

	cmd = exec.Command("git", "tag", v)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "push", "origin", "--tags")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("goreleaser", "--rm-dist")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
})
