package lyhook

import (
	"runtime"
	"strings"
)

const (
	maximumCallerDepth int = 25

	DefaultCallerSkip = 8
)

type CallerFunc func() *runtime.Frame

type Caller interface {
	Frame() *runtime.Frame
}

func NewConstCaller(skip int) *ConstCaller {
	return &ConstCaller{skip: skip}
}

type ConstCaller struct {
	skip int
}

func (c *ConstCaller) Frame() *runtime.Frame {
	rpc := make([]uintptr, 1)
	n := runtime.Callers(c.skip+1, rpc[:])
	if n < 1 {
		return nil
	}
	frame, _ := runtime.CallersFrames(rpc).Next()
	return &frame
}

func NewFuncCaller(fn CallerFunc) *FuncCaller {
	return &FuncCaller{fn: fn}
}

type FuncCaller struct {
	fn CallerFunc
}

func (c *FuncCaller) Frame() *runtime.Frame {
	return c.fn()
}

func WrappedCallerFuncWithPrefix(prefix string) CallerFunc {
	return func() *runtime.Frame {
		pcs := make([]uintptr, maximumCallerDepth)
		_ = runtime.Callers(0, pcs)

		// dynamic get the package name and the minimum caller depth
		var skip int
		for i := 0; i < maximumCallerDepth; i++ {
			name := runtime.FuncForPC(pcs[i]).Name()
			if strings.HasPrefix(name, prefix) {
				skip = i
				break
			}
		}
		return NewConstCaller(skip).Frame()
	}
}
