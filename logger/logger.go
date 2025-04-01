package logger

import (
	"encoding/json"
	"io"
	stdlog "log"
	"os"
	"strings"

	"github.com/sirupsen/logrus" // use logrus for our 3rd party (level) logger but encapsulate its usage here
)

//-------------------------------------------------------------------------------------------------

const (
	logLevelEnvVar  = "LOG_LEVEL"
	logFormatEnvVar = "LOG_FORMAT"
	timestampFormat = "2006-01-02T15:04:05Z07:00"
)

//-------------------------------------------------------------------------------------------------

// Level specifies a logging level
type Level uint32

// Level values
const (
	ErrorLevel Level = Level(logrus.ErrorLevel)
	WarnLevel  Level = Level(logrus.WarnLevel)
	InfoLevel  Level = Level(logrus.InfoLevel)
	DebugLevel Level = Level(logrus.DebugLevel)
)

// LevelString returns the string label for a given level
func (l Level) String() string {
	switch l {
	case ErrorLevel:
		return logrus.ErrorLevel.String()
	case WarnLevel:
		return logrus.WarnLevel.String()
	case InfoLevel:
		return logrus.InfoLevel.String()
	case DebugLevel:
		return logrus.DebugLevel.String()
	default:
		return "unknown"
	}
}

// ParseLevel returns a log Level parsed from a string
func ParseLevel(level string) Level {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return InfoLevel
	}
	return Level(lvl)
}

// Format specifies a logging format
type Format string

// Format values
const (
	JSONFormat Format = "json"
	TextFormat Format = "text"
)

//-------------------------------------------------------------------------------------------------

// Fields are key/value pairs that can be used to enhance log entries using the WithFields() method
type Fields map[string]interface{}

//-------------------------------------------------------------------------------------------------

// Entry is a prepared log entry WithFields() that is ready to be logged
type Entry interface {
	Error(i ...interface{})
	Warn(i ...interface{})
	Info(i ...interface{})
	Debug(i ...interface{})
	Errorf(format string, i ...interface{})
	Warnf(format string, i ...interface{})
	Infof(format string, i ...interface{})
	Debugf(format string, i ...interface{})
	ErrorJSON(item interface{})
	WarnJSON(item interface{})
	InfoJSON(item interface{})
	DebugJSON(item interface{})
	ErrorWithStackTrace(err error)
}

//-------------------------------------------------------------------------------------------------

type logger struct {
	lr     *logrus.Logger
	format Format
}

func (l *logger) GetLevel() Level                        { return Level(l.lr.GetLevel()) }
func (l *logger) SetLevel(level Level)                   { l.lr.SetLevel(logrus.Level(level)) }
func (l *logger) SetOutput(out io.Writer)                { l.lr.SetOutput(out) }
func (l *logger) Error(i ...interface{})                 { l.lr.Error(i...) }
func (l *logger) Warn(i ...interface{})                  { l.lr.Warn(i...) }
func (l *logger) Info(i ...interface{})                  { l.lr.Info(i...) }
func (l *logger) Debug(i ...interface{})                 { l.lr.Debug(i...) }
func (l *logger) Errorf(format string, i ...interface{}) { l.lr.Errorf(format, i...) }
func (l *logger) Warnf(format string, i ...interface{})  { l.lr.Warnf(format, i...) }
func (l *logger) Infof(format string, i ...interface{})  { l.lr.Infof(format, i...) }
func (l *logger) Debugf(format string, i ...interface{}) { l.lr.Debugf(format, i...) }
func (l *logger) ErrorWithStackTrace(err error)          { l.lr.Errorf("%+v", err) }
func (l *logger) ErrorJSON(item interface{})             { l.Error(jsonLog(item)) }
func (l *logger) WarnJSON(item interface{})              { l.Warn(jsonLog(item)) }
func (l *logger) InfoJSON(item interface{})              { l.Info(jsonLog(item)) }
func (l *logger) DebugJSON(item interface{})             { l.Debug(jsonLog(item)) }

//-------------------------------------------------------------------------------------------------

func (l *logger) GetFormat() Format {
	return l.format
}

func (l *logger) SetFormat(format Format) {
	switch format {
	case JSONFormat:
		l.format = JSONFormat
		l.lr.SetFormatter(utcFormatter{&logrus.JSONFormatter{
			TimestampFormat: timestampFormat,
		}})
	default:
		l.format = TextFormat
		l.lr.SetFormatter(utcFormatter{&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: timestampFormat,
		}})
	}
}

type utcFormatter struct {
	logrus.Formatter
}

func (u utcFormatter) Format(e *logrus.Entry) ([]byte, error) {
	e.Time = e.Time.UTC() // force time to always be logged in UTC, independent of what the current system clock timezone is
	return u.Formatter.Format(e)
}

//-------------------------------------------------------------------------------------------------

func (l *logger) WithFields(fields Fields) Entry {
	return &entry{
		le: l.lr.WithFields(logrus.Fields(fields)),
	}
}

//-------------------------------------------------------------------------------------------------

type entry struct {
	le *logrus.Entry
}

func (e *entry) Error(i ...interface{})                 { e.le.Error(i...) }
func (e *entry) Warn(i ...interface{})                  { e.le.Warn(i...) }
func (e *entry) Info(i ...interface{})                  { e.le.Info(i...) }
func (e *entry) Debug(i ...interface{})                 { e.le.Debug(i...) }
func (e *entry) Errorf(format string, i ...interface{}) { e.le.Errorf(format, i...) }
func (e *entry) Warnf(format string, i ...interface{})  { e.le.Warnf(format, i...) }
func (e *entry) Infof(format string, i ...interface{})  { e.le.Infof(format, i...) }
func (e *entry) Debugf(format string, i ...interface{}) { e.le.Debugf(format, i...) }
func (e *entry) ErrorWithStackTrace(err error)          { e.le.Errorf("%+v", err) }
func (e *entry) ErrorJSON(item interface{})             { e.Error(jsonLog(item)) }
func (e *entry) WarnJSON(item interface{})              { e.Warn(jsonLog(item)) }
func (e *entry) InfoJSON(item interface{})              { e.Info(jsonLog(item)) }
func (e *entry) DebugJSON(item interface{})             { e.Debug(jsonLog(item)) }

//-------------------------------------------------------------------------------------------------

var singleton *logger

var currentLevel Level = InfoLevel // default logrus level

// GetLevel returns the current log level
func GetLevel() Level {
	return singleton.GetLevel()
}

// SetLevel sets the current log level
func SetLevel(level Level) {
	currentLevel = level
	singleton.SetLevel(level)
}

// GetFormat returns the current log format
func GetFormat() Format {
	return singleton.GetFormat()
}

// SetFormat sets the current log format
func SetFormat(format Format) {
	singleton.SetFormat(format)
}

// SetOutput overrides the log output stream (used by tests to capture and verify output)
func SetOutput(out io.Writer) {
	singleton.SetOutput(out)
}

// WithFields prepares a log Entry with additional key/value metadata fields
func WithFields(fields Fields) Entry {
	return singleton.WithFields(fields)
}

// Error logs an error message
func Error(i ...interface{}) {
	singleton.Error(i...)
}

// Warn logs a warning message
func Warn(i ...interface{}) {
	singleton.Warn(i...)
}

// Info logs an info message
func Info(i ...interface{}) {
	singleton.Info(i...)
}

// Debug logs a debug message
func Debug(i ...interface{}) {
	singleton.Debug(i...)
}

// Errorf logs a formatted error message
func Errorf(format string, i ...interface{}) { singleton.Errorf(format, i...) }

// Warnf logs a formatted warning message
func Warnf(format string, i ...interface{}) { singleton.Warnf(format, i...) }

// Infof logs a formatted info message
func Infof(format string, i ...interface{}) { singleton.Infof(format, i...) }

// Debugf logs a formatted debug message
func Debugf(format string, i ...interface{}) { singleton.Debugf(format, i...) }

// ErrorJSON logs a serialized JSON object as an error message
func ErrorJSON(item interface{}) {
	singleton.ErrorJSON(item)
}

// WarnJSON logs a serialized JSON object as a warning message
func WarnJSON(item interface{}) {
	singleton.WarnJSON(item)
}

// InfoJSON logs a serialized JSON object as an info message
func InfoJSON(item interface{}) {
	singleton.InfoJSON(item)
}

// DebugJSON logs a serialized JSON object as a debug message
func DebugJSON(item interface{}) {
	singleton.DebugJSON(item)
}

// ErrorWithStackTrace logs an error with a formatted stack trace
func ErrorWithStackTrace(err error) {
	singleton.ErrorWithStackTrace(err)
}

//-------------------------------------------------------------------------------------------------

func init() {
	Reset()
}

// Reset is used to reset the singleton logger object (only exposed for unit testing init behavior)
func Reset() {

	singleton = &logger{lr: logrus.New()}

	singleton.SetFormat(Format(strings.ToLower(os.Getenv(logFormatEnvVar))))
	singleton.SetLevel(ParseLevel(strings.ToLower(os.Getenv(logLevelEnvVar))))

	stdlog.SetOutput(singleton.lr.Writer()) // make stdlog use our writer for output ...
	stdlog.SetFlags(0)                      // ... and turn off its timestamp 'cause logrus will take care of that

}

//-------------------------------------------------------------------------------------------------

func jsonLog(item interface{}) string {
	bytes, _ := json.Marshal(item)
	return string(bytes)
}

//---------------------------------------------------------------------------------------
