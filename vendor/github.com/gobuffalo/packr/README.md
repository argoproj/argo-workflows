# packr

[![GoDoc](https://godoc.org/github.com/gobuffalo/packr?status.svg)](https://godoc.org/github.com/gobuffalo/packr)

Packr is a simple solution for bundling static assets inside of Go binaries. Most importantly it does it in a way that is friendly to developers while they are developing.

## Intro Video

To get an idea of the what and why of packr, please enjoy this short video: [https://vimeo.com/219863271](https://vimeo.com/219863271).

## Installation

```text
$ go get -u github.com/gobuffalo/packr/...
```

## Usage

### In Code

The first step in using Packr is to create a new box. A box represents a folder on disk. Once you have a box you can get `string` or `[]byte` representations of the file.

```go
// set up a new box by giving it a (relative) path to a folder on disk:
box := packr.NewBox("./templates")

// Get the string representation of a file:
html := box.String("index.html")

// Get the string representation of a file, or an error if it doesn't exist:
html, err := box.MustString("index.html")

// Get the []byte representation of a file:
html := box.Bytes("index.html")

// Get the []byte representation of a file, or an error if it doesn't exist:
html, err := box.MustBytes("index.html")
```

### What is a Box?

A box represents a folder, and any sub-folders, on disk that you want to have access to in your binary. When compiling a binary using the `packr` CLI the contents of the folder will be converted into Go files that can be compiled inside of a "standard" go binary. Inside of the compiled binary the files will be read from memory. When working locally the files will be read directly off of disk. This is a seamless switch that doesn't require any special attention on your part.

#### Example

Assume the follow directory structure:

```
├── main.go
└── templates
    ├── admin
    │   └── index.html
    └── index.html
```

The following program will read the `./templates/admin/index.html` file and print it out.

```go
package main

import (
	"fmt"

	"github.com/gobuffalo/packr"
)

func main() {
	box := packr.NewBox("./templates")

	s := box.String("admin/index.html")
	fmt.Println(s)
}
```

### Development Made Easy

In order to get static files into a Go binary, those files must first be converted to Go code. To do that, Packr, ships with a few tools to help build binaries. See below.

During development, however, it is painful to have to keep running a tool to compile those files.

Packr uses the following resolution rules when looking for a file:

1. Look for the file in-memory (inside a Go binary)
1. Look for the file on disk (during development)

Because Packr knows how to fall through to the file system, developers don't need to worry about constantly compiling their static files into a binary. They can work unimpeded.

Packr takes file resolution a step further. When declaring a new box you use a relative path, `./templates`. When Packr recieves this call it calculates out the absolute path to that directory. By doing this it means you can be guaranteed that Packr can find your files correctly, even if you're not running in the directory that the box was created in. This helps with the problem of testing, where Go changes the `pwd` for each package, making relative paths difficult to work with. This is not a problem when using Packr.

---

## Usage with HTTP

A box implements the [`http.FileSystem`](https://golang.org/pkg/net/http/#FileSystemhttps://golang.org/pkg/net/http/#FileSystem) interface, meaning it can be used to serve static files.

```go
package main

import (
	"net/http"

	"github.com/gobuffalo/packr"
)

func main() {
	box := packr.NewBox("./templates")

	http.Handle("/", http.FileServer(box))
	http.ListenAndServe(":3000", nil)
}
```

---

## Building a Binary (the easy way)

When it comes time to build, or install, your Go binary, simply use `packr build` or `packr install` just as you would `go build` or `go install`. All flags for the `go` tool are supported and everything works the way you expect, the only difference is your static assets are now bundled in the generated binary. If you want more control over how this happens, looking at the following section on building binaries (the hard way).

### Building a Binary (the hard way)

Before you build your Go binary, run the `packr` command first. It will look for all the boxes in your code and then generate `.go` files that pack the static files into bytes that can be bundled into the Go binary.

```
$ packr
--> packing foo/foo-packr.go
--> packing example-packr.go
```

Then run your `go build command` like normal.

*NOTE*: It is not recommended to check-in these generated `-packr.go` files. They can be large, and can easily become out of date if not careful. It is recommended that you always run `packr clean` after running the `packr` tool.

#### Cleaning Up

When you're done it is recommended that you run the `packr clean` command. This will remove all of the generated files that Packr created for you.

```
$ packr clean
----> cleaning up example-packr.go
----> cleaning up foo/foo-packr.go
```

Why do you want to do this? Packr first looks to the information stored in these generated files, if the information isn't there it looks to disk. This makes it easy to work with in development.
