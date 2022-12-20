package lyhook

import (
	"runtime"
	"strings"
	"sync"
)

const (
	maximumCallerDepth int = 25

	DefaultCallerSkip = 10
)

var (
	defaultCaller = NewDefaultCaller().SetSkip(DefaultCallerSkip)

	// qualified package name, cached at first use
	hookPackage string

	// Used for caller information initialisation
	callerInitOnce sync.Once
)

type IfCallFrame func(packageName string) bool

type Caller interface {
	Frame() *runtime.Frame
}

func NewDefaultCaller() *DefaultCaller {
	return &DefaultCaller{
		lock: new(sync.Mutex),
	}
}

type DefaultCaller struct {
	skip        int
	ifCallFrame IfCallFrame

	lock *sync.Mutex
}

func (c *DefaultCaller) SetSkip(skip int) *DefaultCaller {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.skip = skip
	return c
}

func (c *DefaultCaller) SetIfCall(fn IfCallFrame) *DefaultCaller {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.ifCallFrame = fn
	return c
}

func (c *DefaultCaller) Frame() *runtime.Frame {
	return getFrame(c.skip, c.ifCallFrame)
}

func getFrame(skip int, fn IfCallFrame) *runtime.Frame {
	pcs := make([]uintptr, maximumCallerDepth)
	_ = runtime.Callers(0, pcs)

	// dynamic get the package name
	callerInitOnce.Do(func() {
		for i := 0; i < maximumCallerDepth; i++ {
			name := runtime.FuncForPC(pcs[i]).Name()
			if strings.Contains(name, "getFrame") {
				hookPackage = getPackageName(name)
				break
			}
		}
	})

	// get skip depth
	if fn != nil {
		for i := 0; i < maximumCallerDepth; i++ {
			name := runtime.FuncForPC(pcs[i]).Name()
			if fn(name) {
				skip = i
				break
			}
		}
	}

	rpc := make([]uintptr, maximumCallerDepth)
	n := runtime.Callers(skip, rpc[:])
	if n < 1 {
		return nil
	}
	frames := runtime.CallersFrames(rpc[:n])

	for frame, next := frames.Next(); next; frame, next = frames.Next() {
		if s := getPackageName(frame.Function); s != hookPackage {
			return &frame
		}
	}
	return nil

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
