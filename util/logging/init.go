package logging

import (
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

//
// The `Init` type is used to collect and emit log messages during application initialization.
// This is useful when you want to log information before your main logger is fully configured.

type initLog struct {
	level   Level
	message string
	fields  Fields
}

type storage struct {
	initLogs []initLog
	mutex    sync.Mutex
	fatal    bool
	out      io.Writer // for testing purposes only
}

var (
	initStorage = &storage{
		initLogs: make([]initLog, 0),
		mutex:    sync.Mutex{},
		fatal:    false,
		out:      os.Stderr,
	}
	setupOnce sync.Once
)

var _ Logger = &initLogger{}

func InitLogger() Logger {
	return initLogger{
		storage: initStorage,
		fields:  make(Fields, 0),
	}
}

func InitLoggerInContext() context.Context {
	return WithLogger(context.Background(), InitLogger())
}

type initLogger struct {
	storage *storage
	fields  Fields
}

func (i initLogger) NewBackgroundContext() context.Context {
	panic("not implemented, don't implement this")
}

func initSignalHandlers() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		// Emit any remaining init logs before exit
		if len(initStorage.initLogs) > 0 {
			emitInitLogs(context.Background(), NewSlogLoggerCustom(Debug, JSON, os.Stderr))
		}
	}()
}

// Caller must hold the lock
func (i initLogger) add(level Level, message string) {
	i.storage.initLogs = append(i.storage.initLogs, initLog{level: level, message: message, fields: i.fields})
	setupOnce.Do(initSignalHandlers)
}

func (i initLogger) Level() Level {
	panic("not implemented, don't implement this")
}

func (i initLogger) WithFatal() Logger {
	i.storage.mutex.Lock()
	defer i.storage.mutex.Unlock()
	i = DeepCopy(i)
	i.storage.fatal = true
	return i
}

func (i initLogger) WithPanic() Logger {
	panic("not implemented, don't implement this")
}

func (i initLogger) WithField(name string, value any) Logger {
	i.storage.mutex.Lock()
	defer i.storage.mutex.Unlock()
	i = DeepCopy(i)
	i.fields[name] = value
	return i
}

func (i initLogger) WithFields(fields Fields) Logger {
	i.storage.mutex.Lock()
	defer i.storage.mutex.Unlock()
	i = DeepCopy(i)
	maps.Copy(fields, i.fields)
	return i
}

func (i initLogger) WithError(err error) Logger {
	i.storage.mutex.Lock()
	defer i.storage.mutex.Unlock()
	i.fields["error"] = err
	return i
}

func (i initLogger) Debugf(ctx context.Context, format string, args ...any) {
	i.Debug(ctx, fmt.Sprintf(format, args...))
}

func (i initLogger) Debug(ctx context.Context, message string) {
	i.storage.mutex.Lock()
	defer i.storage.mutex.Unlock()
	i.add(Debug, message)
}

func (i initLogger) Infof(ctx context.Context, format string, args ...any) {
	i.Info(ctx, fmt.Sprintf(format, args...))
}

func (i initLogger) Info(ctx context.Context, message string) {
	i.storage.mutex.Lock()
	defer i.storage.mutex.Unlock()
	i.add(Info, message)
}

func (i initLogger) Warnf(ctx context.Context, format string, args ...any) {
	i.Warn(ctx, fmt.Sprintf(format, args...))
}

func (i initLogger) Warn(ctx context.Context, message string) {
	i.storage.mutex.Lock()
	defer i.storage.mutex.Unlock()
	i.add(Warn, message)
}

func (i initLogger) Errorf(ctx context.Context, format string, args ...any) {
	i.Error(ctx, fmt.Sprintf(format, args...))
}

func (i initLogger) Error(ctx context.Context, message string) {
	i.storage.mutex.Lock()
	defer i.storage.mutex.Unlock()
	i.add(Error, message)
	if i.storage.fatal {
		// nolint:contextcheck
		emitInitLogs(ctx, NewSlogLoggerCustom(Debug, JSON, i.storage.out))
		if exitFunc := GetExitFunc(); exitFunc != nil {
			exitFunc(1)
		} else {
			os.Exit(1)
		}
	}
}

func emitInitLogs(ctx context.Context, logger Logger) {
	for _, log := range initStorage.initLogs {
		switch log.level {
		case Debug:
			logger.WithFields(log.fields).Debug(ctx, log.message)
		case Info:
			logger.WithFields(log.fields).Info(ctx, log.message)
		case Warn:
			logger.WithFields(log.fields).Warn(ctx, log.message)
		case Error:
			logger.WithFields(log.fields).Error(ctx, log.message)
		}
	}
	initStorage.initLogs = make([]initLog, 0)
}

func DeepCopy(i initLogger) initLogger {
	return initLogger{
		storage: i.storage,
		fields:  maps.Clone(i.fields),
	}
}
