package lyhook

import (
	"io"
	"os"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
)

type RotateFile struct {
	*rotatelogs.RotateLogs
	hasStdout bool
}

func (r *RotateFile) Write(p []byte) (n int, err error) {
	var writer io.Writer

	if r.hasStdout {
		writer = io.MultiWriter(r.RotateLogs, os.Stdout)
	} else {
		writer = r.RotateLogs
	}
	return writer.Write(p)
}

func (r *RotateFile) SetStdout() {
	r.hasStdout = true
}

func NewRotateFile(fpath string) (*RotateFile, error) {
	return NewRotateFileWithTime(fpath, 7*24*time.Hour, 24*time.Hour)
}

func NewRotateFileWithTime(fpath string, maxAge, rotation time.Duration) (*RotateFile, error) {
	f := &RotateFile{}
	var err error
	f.RotateLogs, err = rotatelogs.New(
		fpath+".%Y%m%d%H%M",
		rotatelogs.WithLinkName(fpath),
		rotatelogs.WithMaxAge(maxAge),
		rotatelogs.WithRotationTime(rotation),
	)
	if err != nil {
		return nil, err
	}
	return f, nil
}

type RotateFileMap map[logrus.Level]*RotateFile

func (r RotateFileMap) Close() error {
	var err error
	for _, f := range r {
		if e := f.Close(); err != nil {
			err = e
		}
	}
	return err
}

func NewRotateFileMap(p string) (RotateFileMap, error) {
	rfm := make(RotateFileMap)
	for _, level := range logrus.AllLevels {
		lp := p + "." + level.String()
		rf, err := NewRotateFile(lp)
		if err != nil {
			return nil, err
		}
		rfm[level] = rf
	}
	return rfm, nil
}
