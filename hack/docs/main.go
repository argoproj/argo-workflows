package main

import (
	"os"
)

func main() {
	switch os.Args[1] {
	case "cli":
		generateCLIDocs()
	case "diagram":
		generateDiagram()
	case "fields":
		generateFieldsDocs()
	default:
		panic(os.Args[1])
	}
}
