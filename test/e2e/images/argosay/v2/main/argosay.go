package main

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

func main() {
	err := argosay(os.Args[1:]...)
	if err != nil {
		panic(err)
	}
}
func argosay(args ...string) error {
	if len(args) == 0 {
		args = []string{"echo"}
	}
	switch args[0] {
	case "cat":
		return cat(args[1:])
	case "echo":
		return echo(args[1:])
	case "sleep":
		return sleep(args[1:])
	}
	return errors.New("usage: argosay [cat [file...]|echo [string] [file]|sleep duration]")
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
		err := os.MkdirAll(filepath.Dir(file), 0777)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(file, []byte(args[0]), 0666)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("usage: argosay echo [string] [file]")
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
