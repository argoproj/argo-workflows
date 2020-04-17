package fixtures

import (
	"os"

	log "github.com/sirupsen/logrus"
)

var file = &fileLogger{}

type fileLogger struct {
	file *os.File
}

func (f *fileLogger) Levels() []log.Level {
	return log.AllLevels
}

func (f *fileLogger) Fire(l *log.Entry) error {
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

func (f *fileLogger) Close() error {
	err := f.file.Close()
	f.file = nil
	return err
}

func (f *fileLogger) setFile(file *os.File) error {
	if f.file != nil {
		err := f.Close()
		if err != nil {
			return err
		}
	}
	f.file = file
	return nil
}
