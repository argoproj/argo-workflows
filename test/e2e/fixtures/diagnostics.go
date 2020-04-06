package fixtures

import (
	"os"

	log "github.com/sirupsen/logrus"
)

var diagnostics = &fileHook{}

func init() {
	log.AddHook(diagnostics)
}

type fileHook struct {
	file *os.File
}

func (f *fileHook) Levels() []log.Level {
	return log.AllLevels
}

func (f *fileHook) Fire(l *log.Entry) error {
	if f.file != nil {
		s, err := l.String()
		if err != nil {
			return err
		}
		_, err = f.file.WriteString(s)
		return err
	}
	return nil
}

func (f *fileHook) Close() error {
	err := f.file.Close()
	f.file = nil
	return err
}

func (f *fileHook) setFile(file *os.File) error {
	if f.file != nil {
		err := f.Close()
		if err != nil {
			return err
		}
	}
	f.file = file
	return nil
}
