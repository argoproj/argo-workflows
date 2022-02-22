package emissary

import (
	"bufio"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_multiReaderCloser(t *testing.T) {
	a := ioutil.NopCloser(strings.NewReader("a"))
	b := ioutil.NopCloser(strings.NewReader("b"))
	c := newMultiReaderCloser(a, b)
	s := bufio.NewScanner(c)
	assert.True(t, s.Scan())
	assert.Equal(t, "ab", s.Text())
	assert.False(t, s.Scan())
	assert.NoError(t, c.Close())
}
