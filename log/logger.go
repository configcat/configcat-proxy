package log

import (
	"github.com/configcat/go-sdk/v8"
	"io"
	"log"
	"os"
)

type Level int

type Logger interface {
	GetLevel() configcat.LogLevel // for the SDK

	Level() Level

	WithLevel(level Level) Logger
	WithPrefix(prefix string) Logger

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})

	// Reportf logs regardless of level
	Reportf(format string, args ...interface{})
}

const (
	Debug Level = iota
	Info  Level = iota
	Warn  Level = iota
	Error Level = iota
	None  Level = iota
)

type logger struct {
	level       Level
	errorLogger *log.Logger
	outLogger   *log.Logger
	prefix      string
}

func NewNullLogger() Logger {
	return &logger{level: None}
}

func NewDebugLogger() Logger {
	return &logger{
		level:       Debug,
		errorLogger: log.New(os.Stderr, "", log.Ldate),
		outLogger:   log.New(os.Stdout, "", log.Ldate),
	}
}

func NewLogger(err io.Writer, out io.Writer, level Level) Logger {
	return &logger{
		level:       level,
		errorLogger: log.New(err, "", log.Ldate|log.Ltime|log.LUTC),
		outLogger:   log.New(out, "", log.Ldate|log.Ltime|log.LUTC),
	}
}

func (l *logger) WithLevel(level Level) Logger {
	return &logger{
		level:       level,
		errorLogger: l.errorLogger,
		outLogger:   l.outLogger,
		prefix:      l.prefix,
	}
}

func (l *logger) WithPrefix(prefix string) Logger {
	if l.prefix != "" {
		prefix = l.prefix + "/" + prefix
	}
	return &logger{
		level:       l.level,
		errorLogger: l.errorLogger,
		outLogger:   l.outLogger,
		prefix:      prefix,
	}
}

func (l *logger) GetLevel() configcat.LogLevel {
	switch l.level {
	case Debug:
		return configcat.LogLevelDebug
	case Info:
		return configcat.LogLevelInfo
	case Warn:
		return configcat.LogLevelWarn
	case Error:
		return configcat.LogLevelError
	default:
		return configcat.LogLevelWarn
	}
}

func (l *logger) Level() Level {
	return l.level
}

func (l *logger) Debugf(format string, values ...interface{}) {
	l.logf(Debug, format, values...)
}

func (l *logger) Infof(format string, values ...interface{}) {
	l.logf(Info, format, values...)
}

func (l *logger) Warnf(format string, values ...interface{}) {
	l.logf(Warn, format, values...)
}

func (l *logger) Errorf(format string, values ...interface{}) {
	l.logf(Error, format, values...)
}

func (l *logger) Reportf(format string, values ...interface{}) {
	if l.level == None {
		return
	}
	pref := ""
	if l.prefix != "" {
		pref = "<" + l.prefix + ">"
	}
	if pref == "" {
		l.outLogger.Printf(format, values...)
	} else {
		l.outLogger.Printf(pref+" "+format, values...)
	}
}

func (l *logger) logf(level Level, format string, values ...interface{}) {
	if level >= l.level {
		var lo *log.Logger
		if level == Error {
			lo = l.errorLogger
		} else {
			lo = l.outLogger
		}
		pref := ""
		if l.prefix != "" {
			pref = " <" + l.prefix + ">"
		}
		lo.Printf(level.prefix()+pref+" "+format, values...)
	}
}

func (level Level) prefix() string {
	switch level {
	case Debug:
		return "[debug]"
	case Info:
		return "[info]"
	case Warn:
		return "[warning]"
	case Error:
		return "[error]"
	}
	return "-"
}
