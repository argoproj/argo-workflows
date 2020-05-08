package main

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

const usage = "argosay [cat file|echo message (file)|sleep duration]"

func main() {
	switch len(os.Args) {
	case 0:
		println("hello argo")
		switch os.Args[1] {
		case "cat":
			switch len(os.Args) {
			case 2:
				open, err := os.Open(os.Args[2])
				maybePanic(err)
				_, err = io.Copy(os.Stdout, open)
				maybePanic(err)
			default:
				fatal()
			}
		case "echo":
			switch len(os.Args) {
			case 1:
				println("hello argo")
			case 2:
				println(os.Args[2])
			case 3:
				file := os.Args[2]
				maybePanic(os.MkdirAll(file, 0777))
				maybePanic(ioutil.WriteFile(file, []byte(os.Args[1]), 0666))
			default:
				fatal()
			}
		case "sleep":
			switch len(os.Args) {
			case 2:
				duration, err := time.ParseDuration(os.Args[2])
				maybePanic(err)
				time.Sleep(duration)
			default:
				fatal()
			}
		default:
			fatal()
		}
	}
}

func fatal() {
	println(strings.Join(os.Args, " "))
	panic(usage)
}

func maybePanic(err error) {
	if err != nil {
		panic(err)
	}
}
