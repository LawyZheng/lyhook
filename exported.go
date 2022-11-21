package lyhook

import (
	"io"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	hook = NewLyHook(io.Discard, devFormatter)
	lock = new(sync.Mutex)
)

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

func Add(module string, newhook *LyHook) *logrus.Entry {
	return hook.Add(module, newhook)
}

func Apply(logger *logrus.Logger) {
	hook.Apply(logger)
}

func SetFormatter(formatter logrus.Formatter) {
	hook.SetFormatter(formatter)
}

func GetFormatter() logrus.Formatter {
	return hook.GetFormatter()
}

func SetDefaultWriter(defaultWriter io.Writer) {
	hook.SetDefaultWriter(defaultWriter)
}

func init() {
	hook.Apply(logrus.StandardLogger())
}
