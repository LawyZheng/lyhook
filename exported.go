package lyhook

import (
	"io"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	hook = NewLyHook(io.Discard, devFormatter).
		SetCaller(NewDefaultCaller().SetSkip(DefaultCallerSkip + 2))
	lock = new(sync.Mutex)
)

type Logger interface {
	logrus.FieldLogger

	Trace(args ...interface{})
	Tracef(format string, args ...interface{})
}

func SetHook(h *LyHook) {
	lock.Lock()
	defer lock.Unlock()
	hook = h
}

func PickFormatter(isdev bool) logrus.Formatter {
	if isdev {
		return devFormatter
	}
	return defaultFormatter
}

func Add(module string, newhook *LyHook) Logger {
	return hook.Add(module, newhook)
}

func Apply(logger *logrus.Logger) *LyHook {
	return hook.Apply(logger)
}

func SetCaller(caller Caller) *LyHook {
	return hook.SetCaller(caller)
}

func SetFormatter(formatter logrus.Formatter) *LyHook {
	return hook.SetFormatter(formatter)
}

func GetFormatter() logrus.Formatter {
	return hook.GetFormatter()
}

func SetDefaultWriter(defaultWriter io.Writer) *LyHook {
	return hook.SetDefaultWriter(defaultWriter)
}

func NewLoggerWithHook(h *LyHook) *logrus.Logger {
	logger := logrus.New()
	logger.SetReportCaller(logrus.StandardLogger().ReportCaller)
	logger.SetFormatter(logrus.StandardLogger().Formatter)
	logger.SetOutput(logrus.StandardLogger().Out)
	logger.SetLevel(logrus.GetLevel())
	h.Apply(logger)
	return logger
}

func init() {
	hook.Apply(logrus.StandardLogger())
}
