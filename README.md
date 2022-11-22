# A Easy Hook for Logrus

[![GoDoc](https://godoc.org/github.com/layzheng/lyhook?status.svg)](https://pkg.go.dev/github.com/lawyzheng/lyhook)

Sometimes developers like to write directly to a file on the filesystem. This is a hook for [`logrus`](https://github.com/sirupsen/logrus) which designed to allow users to do that. The log levels are dynamic at instantiation of the hook, so it is capable of logging at some or all levels.

## Normal Example
```go
import (
	_ "github.com/lawyzheng/lyhook"
	"github.com/sirupsen/logrus"
)

func main(){
	// this is a normal use, it functions the same as logrus do
	logrus.Info("this is a log")
}
```

## Rotate File Example

```go
import (
	"github.com/lawyzheng/lyhook"
	"github.com/sirupsen/logrus"
)

func main(){
	// this will regist a normal rotation use with rotating every 24 hours, retaining last one week log
	wr, err := lyhook.NewRotateFile("mylog.log")
	if err != nil {
		panic(err)
	}
	lyhook.SetDefaultWriter(wr)

	// this will trigger logrus and the hook
	logrus.Info("this is a log")
}
```

## Rotate File With Different Level Example

```go
import (
	"github.com/lawyzheng/lyhook"
	"github.com/sirupsen/logrus"
)

func main(){
	// this will regist a normal rotation use with rotating every 24 hours, retaining last one week log
	wrs, err := lyhook.NewRotateFileMap("mylog.log")
	if err != nil {
		panic(err)
	}

	hook := lyhook.NewLyHook(hook, lyhook.GetFormatter())

	hook.Apply(logrus.StandardLogger())

	// this is trigger logrus and the hook
	logrus.Debug("this is a debug log") // write to debug log file
	logrus.Info("this is a info log") // write to info log file
}
```

### Formatters
`lyhook` will not strip colors from any `TextFormatter` type formatters when writing to local file. Make sure you need to strip colors when log to file, because colorful text will not format prettified.

If no formatter is provided via `lyhook.NewLyHook`, a default colorful text formatter will be used.


### Note:
User who run the go application must have read/write permissions to the selected log files. If the files do not exists yet, then user must have permission to the target directory.
