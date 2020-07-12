package main

import (
	"time"

	"go.uber.org/zap"
)

type CheckNotifier interface {
	Notify(checkName string, timestamp time.Time, ok bool, reason string) error
}

type LogNotifier struct{}

func (LogNotifier) Notify(checkName string, timestamp time.Time, ok bool, reason string) error {
	l := zap.L().With(zap.String("name", checkName), zap.Time("time", timestamp), zap.String("reason", reason))
	if ok {
		l.Info("Check back to normal.")
	} else {
		l.Warn("Check failed.")
	}
	return nil
}
