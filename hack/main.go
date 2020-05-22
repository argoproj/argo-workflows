package main

import "os"

func main() {
	switch os.Args[1] {
	case "docgen":
		generateDocs()
	case "gencrds":
		genCRDs()
	case "genschemaassets":
		genSchemaAssets()
	case "kubeifyswagger":
		kubeifySwagger(os.Args[2], os.Args[3])
	case "secondaryswaggergen":
		secondarySwaggerGen()
	case "readmegen":
		readmeGen()
	default:
		panic(os.Args[1])
	}
}
