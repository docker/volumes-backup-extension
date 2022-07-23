package log

import (
	"github.com/sirupsen/logrus"
	"io"
)

var logger = logrus.New()

func SetOutput(w io.Writer) {
	logger.Out = w
}

func Info(args ...interface{}) {
	logger.Info(args...)
}

func Infof(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

func Error(args ...interface{}) {
	logger.Error(args...)
}