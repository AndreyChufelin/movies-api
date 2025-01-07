package logger

import (
	"log/slog"
)

type Logger struct {
	*slog.Logger
}

type Exit struct{ Code int }

func (cl *Logger) Fatal(msg string, args ...interface{}) {
	cl.Error(msg, args...)
	panic(Exit{1})
}
