package insight_server

import "github.com/Sirupsen/logrus"

// Sets up the logging to use UTC timestamps and a proper output format
func SetupLogging() {
	if logrus.IsTerminal() {
		logrus.SetFormatter(&logrus.TextFormatter{
			ForceColors: true,
		})
	} else {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	logrus.SetLevel(logrus.InfoLevel)

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
