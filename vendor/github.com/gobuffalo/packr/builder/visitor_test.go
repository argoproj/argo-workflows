package builder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Visitor(t *testing.T) {
	r := require.New(t)
	v := newVisitor("../example/example.go")
	r.NoError(v.Run())

	r.Equal("example", v.Package)
	r.Len(v.Errors, 0)
	r.Len(v.Boxes, 5)
	r.Equal([]string{"./assets", "./bar", "./foo", "./sf", "./templates"}, v.Boxes)
}
