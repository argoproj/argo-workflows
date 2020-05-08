package main

import "os"

func main() {
	switch len(os.Args) {
	case 1:
		println("hello argo")
	default:
		panic(os.Args)
	}
}
