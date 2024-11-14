package main

import (
	"os"
)

func main() {
	switch os.Args[1] {
	case "cleancrd":
		cleanCRD(os.Args[2])
	case "minimizecrd":
		minimizeCRD(os.Args[2])
	default:
		panic(os.Args[1])
	}
}
