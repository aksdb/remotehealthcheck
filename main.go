package main

import (
	"crypto/tls"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
)

var (
	checkInterval = flag.String("interval", "10m", "Interval to run checks in.")
)

type Check interface {
	Perform() bool
}

type BaseCheck struct {
	Name            string `yaml:"name"`
	Type            string `yaml:"type"`
	lastCheckTime   time.Time
	lastStateChange time.Time
	lastOk          bool
	notifier        CheckNotifier
}

func (b *BaseCheck) UpdateState(ok bool, reason string) {
	skipNotify := b.lastStateChange.IsZero()

	b.lastCheckTime = time.Now()
	if ok != b.lastOk {
		b.lastStateChange = time.Now()
		b.lastOk = ok
		if !skipNotify {
			if err := b.notifier.Notify(b.Name, b.lastStateChange, ok, reason); err != nil {
				zap.L().Error("Cannot notify.", zap.String("name", b.Name), zap.Error(err))
			}
		}
	}
}

type Group struct {
	*BaseCheck
	Checks []Check
}

func (g *Group) Perform() bool {
	allOk := true
	for _, c := range g.Checks {
		if !c.Perform() {
			allOk = false
		}
	}
	g.UpdateState(allOk, "")
	return allOk
}

type TlsCheck struct {
	*BaseCheck
	Address  string `yaml:"address"`
	Insecure bool   `yaml:"insecure"`
}

func (t *TlsCheck) Perform() bool {
	conn, err := tls.Dial("tcp", t.Address, &tls.Config{
		InsecureSkipVerify: t.Insecure,
	})
	if err != nil {
		t.UpdateState(false, err.Error())
		return false
	} else {
		t.UpdateState(true, "")
		conn.Close()
		return true
	}
}

type rawCheck map[interface{}]interface{}

func main() {
	flag.Parse()
	initLogger()

	checkIntervalDuration, err := time.ParseDuration(*checkInterval)
	if err != nil {
		zap.L().Fatal("Cannot parse interval.", zap.Error(err))
	}

	f, err := os.Open("checks.yaml")
	if err != nil {
		zap.L().Fatal("Cannot open check definitions.", zap.Error(err))
	}

	var rawChecks []rawCheck
	if err := yaml.NewDecoder(f).Decode(&rawChecks); err != nil {
		zap.L().Fatal("Cannot decode yaml structure.", zap.Error(err))
	}

	checks := mapRawChecks(rawChecks)

	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, os.Interrupt, syscall.SIGTERM)

mainloop:
	for {
		zap.L().Info("Running checks.")
		for _, c := range checks {
			c.Perform()
		}

		select {
		case <-time.After(checkIntervalDuration):
			continue
		case <-exitChan:
			zap.L().Info("Exiting.")
			break mainloop
		}
	}
}

func mapRawChecks(rawChecks []rawCheck) []Check {
	var checks []Check
	for _, rawCheck := range rawChecks {
		baseCheck := &BaseCheck{
			Name:     rawCheck.StringValue("name"),
			Type:     rawCheck.StringValue("type"),
			notifier: LogNotifier{},
		}

		var check Check

		switch baseCheck.Type {
		case "group":
			check = &Group{
				BaseCheck: baseCheck,
				Checks:    mapRawChecks(rawCheck.SubChecks("checks")),
			}
		case "tls":
			check = &TlsCheck{
				BaseCheck: baseCheck,
				Address:   rawCheck.StringValue("address"),
				Insecure:  rawCheck.BoolValue("insecure"),
			}
		default:
			zap.L().Fatal("Unrecognized type.", zap.String("type", baseCheck.Type))
		}

		checks = append(checks, check)
	}

	return checks
}

func (r rawCheck) StringValue(name string) string {
	raw, exists := r[name]
	if !exists {
		return ""
	}
	val, ok := raw.(string)
	if !ok {
		zap.L().Fatal("Cannot read value as string.", zap.String("name", name))
	}
	return val
}

func (r rawCheck) BoolValue(name string) bool {
	raw, exists := r[name]
	if !exists {
		return false
	}
	val, ok := raw.(bool)
	if !ok {
		zap.L().Fatal("Cannot read value as bool.", zap.String("name", name))
	}
	return val
}

func (r rawCheck) SubChecks(name string) []rawCheck {
	raw, exists := r[name]
	if !exists {
		return []rawCheck{}
	}
	val, ok := raw.([]rawCheck)
	if !ok {
		intVal, ok := raw.([]interface{})
		if !ok {
			zap.L().Fatal("Cannot read value as list of checks.", zap.String("name", name))
		}
		val = make([]rawCheck, len(intVal))
		for i := range intVal {
			val[i], ok = intVal[i].(rawCheck)
			if !ok {
				zap.L().Fatal("Cannot read value as list of checks.", zap.String("name", name))
			}
		}
	}
	return val
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
