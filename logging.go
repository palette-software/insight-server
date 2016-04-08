package insight_server

import (
	"fmt"

	"github.com/Sirupsen/logrus"
)

// Sets up the logging to use UTC timestamps and a proper output format
func SetupLogging(logFormat, logLevel string) {
	switch logFormat {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	case "text":
		logrus.SetFormatter(&logrus.TextFormatter{})
	// force colors
	case "color":
		logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true})
	default:
		panic(fmt.Sprintf("Cannot find proper log format for %s", logFormat))
	}

	switch logLevel {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	// add the UTC timestamp hook
	logrus.AddHook(new(LogrusTimeFixerHook))
}

// UTC timestamp fixer hook
// ------------------------

type LogrusTimeFixerHook struct{}

func (l *LogrusTimeFixerHook) Fire(entry *logrus.Entry) error {
	entry.Time = entry.Time.UTC()
	return nil
}

func (l *LogrusTimeFixerHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel, logrus.WarnLevel, logrus.InfoLevel, logrus.DebugLevel}
}
