package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	err := argosay(os.Args[1:]...)
	if err != nil {
		if exitErr, ok := err.(exitError); ok {
			os.Exit(exitErr.code)
		}
		panic(err)
	}
}
func argosay(args ...string) error {
	if len(args) == 0 {
		args = []string{"echo"}
	}
	switch args[0] {
	case "assert_contains":
		return assertContains(args[1:])
	case "cat":
		return cat(args[1:])
	case "echo":
		return echo(args[1:])
	case "exit":
		return exit(args[1:])
	case "sleep":
		return sleep(args[1:])
	}
	return errors.New("usage: argosay [assert_contains file string|cat [file...]|echo [string] [file]|sleep duration|exit [code]]")
}

func assertContains(args []string) error {
	switch len(args) {
	case 2:
		filename := args[0]
		substr := args[1]
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		if !strings.Contains(string(data), substr) {
			return fmt.Errorf(`expected "%s" to contain "%s", but was "%s"`, filename, substr, string(data))
		}
		return nil
	}
	return errors.New("usage: assert_contains data string")
}

func cat(args []string) error {
	for _, file := range args {
		open, err := os.Open(file)
		if err != nil {
			return err
		}
		_, err = io.Copy(os.Stdout, open)
		if err != nil {
			return err
		}
	}
	return nil
}

func echo(args []string) error {
	switch len(args) {
	case 0:
		println("hello argo")
		return nil
	case 1:
		println(args[0])
		return nil
	case 2:
		file := args[1]
		dir := filepath.Dir(file)
		_, err := os.Stat(dir)
		if os.IsNotExist(err) {
			err := os.MkdirAll(dir, 0777)
			if err != nil {
				return err
			}
		}
		err = ioutil.WriteFile(file, []byte(args[0]), 0666)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("usage: argosay echo [string] [file]")
}

type exitError struct {
	code int
}

func (e exitError) Error() string {
	return fmt.Sprintf("exit code %v", e.code)
}

func exit(args []string) error {
	switch len(args) {
	case 1:
		code, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		if code != 0 {
			return exitError{code}
		}
	}
	return nil
}

func sleep(args []string) error {
	switch len(args) {
	case 1:
		duration, err := time.ParseDuration(args[0])
		if err != nil {
			return err
		}
		time.Sleep(duration)
		return nil
	}
	return errors.New("usage: argosay sleep duration")
}
