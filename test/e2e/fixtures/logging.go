package fixtures

import (
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
)

var stdout = &stdoutHook{}
var file = &fileLogger{}

func init() {
	log.SetOutput(ioutil.Discard)
	log.SetLevel(log.DebugLevel)
	log.AddHook(stdout)
	log.AddHook(file)
	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
}

type stdoutHook struct {
}

func (s stdoutHook) Levels() []log.Level {
	return []log.Level{log.PanicLevel, log.FatalLevel, log.ErrorLevel, log.WarnLevel, log.InfoLevel}
}

func (s stdoutHook) Fire(entry *log.Entry) error {
	str, err := entry.String()
	if err != nil {
		return err
	}
	_, err = os.Stdout.WriteString(str)
	return err
}

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
