package main

import (
	"time"

	"go.uber.org/zap"
)

type CheckState struct {
	Timestamp time.Time
	Ok        bool
	Reason    string
}

type CheckNotifier interface {
	Notify(checkName string, state CheckState) error
}

type MultiNotifier struct {
	notifiers []CheckNotifier
}

func (m *MultiNotifier) Notify(checkName string, state CheckState) error {
	for _, n := range m.notifiers {
		if err := n.Notify(checkName, state); err != nil {
			zap.L().Error("Could not call notifier.", zap.Error(err))
		}
	}
	return nil
}

func (m *MultiNotifier) RegisterNotifier(cn CheckNotifier) {
	m.notifiers = append(m.notifiers, cn)
}

type LogNotifier struct{}

func (LogNotifier) Notify(checkName string, state CheckState) error {
	l := zap.L().With(zap.String("name", checkName), zap.Time("time", state.Timestamp), zap.String("reason", state.Reason))
	if state.Ok {
		l.Info("Check succeeded.")
	} else {
		l.Warn("Check failed.")
	}
	return nil
}
