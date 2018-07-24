package example

import (
	"github.com/gobuffalo/packr"
)

var a = packr.NewBox("./foo")

type S struct{}

func (S) f(packr.Box) {}

func init() {
	// packr.NewBox("../idontexists")

	b := "./baz"
	packr.NewBox(b) // won't work, no variables allowed, only strings

	foo("/templates", packr.NewBox("./templates"))
	packr.NewBox("./assets")

	packr.NewBox("./bar")

	s := S{}
	s.f(packr.NewBox("./sf"))
}

func foo(s string, box packr.Box) {}
