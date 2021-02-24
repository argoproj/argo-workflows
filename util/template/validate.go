package template

import (
	"io"
	"io/ioutil"

	"github.com/valyala/fasttemplate"
)

func Validate(s string, f func(tag string) error) error {
	t, err := fasttemplate.NewTemplate(s, prefix, suffix)
	if err != nil {
		return err
	}
	_, err = t.ExecuteFunc(ioutil.Discard, func(w io.Writer, tag string) (int, error) {
		kind, _ := parseTag(tag)
		switch kind {
		case kindExpression:
			return 0, nil // we do not validate expression templates
		default:
			return 0, f(tag)
		}
	})
	return err
}
