package main

import (
	"fmt"
	"os"
)

func main() {
	switch os.Args[1] {
	case "cleancrd":
		err := cleanCRD(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, os.Args[2], "error:", err)
			//os.Exit(1)
		}
	case "minimizecrd":
		err := minimizeCRD(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, os.Args[2], "error:", err)
			//os.Exit(1)
		}
	default:
		panic(os.Args[1])
	}
}
