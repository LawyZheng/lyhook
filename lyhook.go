// Package lfshook is hook for sirupsen/logrus that used for writing the logs to local files.
package lyhook

import (
	"context"
	"fmt"
	"io"
	"log"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	minimumCallerDepth int
	lyhookPackage      string
	callerInitOnce     sync.Once
)

const (
	maximumCallerDepth int = 30
	knownLogrusFrames  int = 11
)

type CtxKey string

const (
	ctxKeyName CtxKey = "moduleName"
)

// We are logging to file, strip colors to make the output more readable.
var defaultFormatter = &logrus.TextFormatter{DisableColors: true}
var devFormatter = &logrus.TextFormatter{ForceColors: true, FullTimestamp: true}

// WriterMap is map for mapping a log level to an io.Writer.
// Multiple levels may share a writer, but multiple writers may not be used for one level.
type WriterMap map[logrus.Level]io.Writer

// LyHook is a hook to handle writing to local log files.
type LyHook struct {
	writers   WriterMap
	levels    []logrus.Level
	lock      *sync.Mutex
	formatter logrus.Formatter

	defaultWriter    io.Writer
	hasDefaultWriter bool

	logger        *logrus.Logger
	loggerApplied bool
	hookMap       map[string]*LyHook
}

// NewHook returns new LFS hook.
// Output can be a string, io.Writer, WriterMap or PathMap.
// If using io.Writer or WriterMap, user is responsible for closing the used io.Writer.
func NewLyHook(output interface{}, formatter logrus.Formatter) *LyHook {
	hook := &LyHook{
		lock:    new(sync.Mutex),
		hookMap: make(map[string]*LyHook),
	}

	hook.SetFormatter(formatter)

	switch output := output.(type) {
	case io.Writer:
		hook.SetDefaultWriter(output)
	case WriterMap:
		hook.writers = output
		for level := range output {
			hook.levels = append(hook.levels, level)
		}
	case RotateFileMap:
		hook.writers = make(WriterMap)
		for level, f := range output {
			hook.levels = append(hook.levels, level)
			hook.writers[level] = f
		}
	default:
		panic(fmt.Sprintf("unsupported level map type: %v", reflect.TypeOf(output)))
	}

	return hook
}

func (hook *LyHook) Apply(logger *logrus.Logger) {
	hook.lock.Lock()
	defer hook.lock.Unlock()

	logger.AddHook(hook)
	hook.logger = logger
	hook.loggerApplied = true
}

func (hook *LyHook) Add(module string, newhook *LyHook) *logrus.Entry {
	var (
		logger *logrus.Logger
		ctx    = context.Background()
	)

	if newhook == nil {
		newhook = hook
	}

	hook.lock.Lock()
	defer hook.lock.Unlock()

	hook.hookMap[module] = newhook

	if hook.loggerApplied {
		logger = hook.logger
	} else {
		logger = logrus.StandardLogger()
	}

	ctx = context.WithValue(ctx, ctxKeyName, module)
	return logger.WithContext(ctx)
}

// SetFormatter sets the format that will be used by hook.
// If using text formatter, this method will disable color output to make the log file more readable.
func (hook *LyHook) SetFormatter(formatter logrus.Formatter) {
	hook.lock.Lock()
	defer hook.lock.Unlock()
	if formatter == nil {
		formatter = defaultFormatter
	}
	hook.formatter = formatter
}

// SetDefaultWriter sets default writer for levels that don't have any defined writer.
func (hook *LyHook) SetDefaultWriter(defaultWriter io.Writer) {
	hook.lock.Lock()
	defer hook.lock.Unlock()
	hook.defaultWriter = defaultWriter
	hook.hasDefaultWriter = true
}

func (hook *LyHook) GetFormatter() logrus.Formatter {
	hook.lock.Lock()
	defer hook.lock.Unlock()
	return hook.formatter
}

// Fire writes the log file to defined path or using the defined writer.
// User who run this function needs write permissions to the file or directory if the file does not yet exist.
func (hook *LyHook) Fire(entry *logrus.Entry) error {
	h := hook.findHook(entry.Context)

	h.lock.Lock()
	defer h.lock.Unlock()
	if h.writers != nil || h.hasDefaultWriter {
		return h.ioWrite(entry)
	}

	return nil
}

func (hook *LyHook) findHook(ctx context.Context) *LyHook {
	if ctx == nil {
		return hook
	}

	val := ctx.Value(ctxKeyName)
	switch val := val.(type) {
	case nil:
		return hook
	case string:
		h := hook.hookMap[val]
		if h != nil {
			// TODO: to recurise to find hook
			// return h.findHook(ctx)
			return h
		}
		return hook
	default:
		fmt.Printf("unsupported context value type: %v", reflect.TypeOf(val))
		return hook
	}
}

// Write a log line to an io.Writer.
func (hook *LyHook) ioWrite(entry *logrus.Entry) error {
	var (
		writer io.Writer
		msg    []byte
		err    error
		ok     bool
	)

	if writer, ok = hook.writers[entry.Level]; !ok {
		if hook.hasDefaultWriter {
			writer = hook.defaultWriter
		} else {
			return nil
		}
	}

	if level := entry.Level; level <= logrus.ErrorLevel {
		// pc, _, line, _ := runtime.Caller(8)
		// entry.Data["func"] = runtime.FuncForPC(pc).Name()
		// entry.Data["line"] = line
		if caller := getCaller(); caller != nil {
			entry.Data["func"] = caller.Function
			entry.Data["line"] = caller.Line
		}
	}

	// use our formatter instead of entry.String()
	msg, err = hook.formatter.Format(entry)

	if err != nil {
		log.Println("failed to generate string for entry:", err)
		return err
	}
	_, err = writer.Write(msg)
	return err
}

// Levels returns configured log levels.
func (hook *LyHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// getPackageName reduces a fully qualified function name to the package name
// There really ought to be to be a better way...
func getPackageName(f string) string {
	for {
		lastPeriod := strings.LastIndex(f, ".")
		lastSlash := strings.LastIndex(f, "/")
		if lastPeriod > lastSlash {
			f = f[:lastPeriod]
		} else {
			break
		}
	}

	return f
}

// getCaller retrieves the name of the first non-logrus calling function
func getCaller() *runtime.Frame {
	// cache this package's fully-qualified name
	callerInitOnce.Do(func() {
		pcs := make([]uintptr, maximumCallerDepth)
		_ = runtime.Callers(0, pcs)

		// dynamic get the package name and the minimum caller depth
		for i := 0; i < maximumCallerDepth; i++ {
			funcName := runtime.FuncForPC(pcs[i]).Name()
			if strings.Contains(funcName, "getCaller") {
				lyhookPackage = getPackageName(funcName)
				break
			}
		}

		minimumCallerDepth = knownLogrusFrames
	})

	// Restrict the lookback frames to avoid runaway lookups
	pcs := make([]uintptr, maximumCallerDepth)
	depth := runtime.Callers(minimumCallerDepth, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	for f, again := frames.Next(); again; f, again = frames.Next() {
		pkg := getPackageName(f.Function)

		// If the caller isn't part of this package, we're done
		if pkg != lyhookPackage {
			return &f //nolint:scopelint
		}
	}

	// if we got here, we failed to find the caller's context
	return nil
}

func init() {
	minimumCallerDepth = 1
}
