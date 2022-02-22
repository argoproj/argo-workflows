package emissary

import (
	"io"
)

type multiReaderCloser struct {
	io.Reader
	closer []io.ReadCloser
}

func newMultiReaderCloser(x ...io.ReadCloser) io.ReadCloser {
	var readers []io.Reader
	for _, r := range x {
		readers = append(readers, r)
	}
	return &multiReaderCloser{
		Reader: io.MultiReader(readers...),
		closer: x,
	}
}

func (m *multiReaderCloser) Close() error {
	for _, c := range m.closer {
		if err := c.Close(); err != nil {
			return err
		}
	}
	return nil
}
