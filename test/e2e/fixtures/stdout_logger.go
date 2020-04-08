package fixtures

import (
	"os"

	log "github.com/sirupsen/logrus"
)

var stdout = &stdoutLogger{}

type stdoutLogger struct {
}

func (s stdoutLogger) Levels() []log.Level {
	return []log.Level{log.PanicLevel, log.FatalLevel, log.ErrorLevel, log.WarnLevel, log.InfoLevel}
}

func (s stdoutLogger) Fire(entry *log.Entry) error {
	str, err := entry.String()
	if err != nil {
		return err
	}
	_, err = os.Stdout.WriteString(str)
	return err
}
