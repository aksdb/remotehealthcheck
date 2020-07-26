package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	checksFile    = flag.String("checks", "checks.yaml", "File defining the checks to be run.")
	checkInterval = flag.String("interval", "10m", "Interval to run checks in.")
	webListener   = flag.String("webListener", "", "Address to listen on for web based status queries.")
	dontLogChecks = flag.Bool("dontLogChecks", false, "Don't log check changes.")
)

func main() {
	flag.Parse()
	initLogger()

	sm := NewSubroutineManager()

	notifier := &MultiNotifier{}

	if !*dontLogChecks {
		notifier.RegisterNotifier(&LogNotifier{})
	}

	if *webListener != "" {
		notifier.RegisterNotifier(NewWebNotifier(sm, *webListener))
	}

	hc := NewHealthChecker(*checksFile, notifier)

	checkIntervalDuration, err := time.ParseDuration(*checkInterval)
	if err != nil {
		zap.L().Fatal("Cannot parse interval.", zap.Error(err))
	}

	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, os.Interrupt, syscall.SIGTERM)

mainloop:
	for {
		hc.PerformChecks()

		select {
		case <-time.After(checkIntervalDuration):
			continue
		case <-exitChan:
			zap.L().Info("Exiting.")
			break mainloop
		}
	}

	sm.Shutdown()
}

func initLogger() {
	zc := zap.NewProductionConfig()
	zc.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	l, err := zc.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(l)
}

type SubroutineManager struct {
	sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewSubroutineManager() *SubroutineManager {
	sm := &SubroutineManager{}

	sm.ctx, sm.cancel = context.WithCancel(context.Background())

	return sm
}

func (sm *SubroutineManager) Context() context.Context {
	return sm.ctx
}

func (sm *SubroutineManager) Shutdown() {
	sm.cancel()
	sm.Wait()
}
