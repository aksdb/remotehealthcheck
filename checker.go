package main

import (
	"os"
	"regexp"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type HealthChecker struct {
	checks   []Check
	notifier CheckNotifier
}

func NewHealthChecker(checkFilepath string, notifier CheckNotifier) *HealthChecker {
	f, err := os.Open(checkFilepath)
	if err != nil {
		zap.L().Fatal("Cannot open check definitions.", zap.Error(err))
	}

	var rawChecks []rawCheck
	if err := yaml.NewDecoder(f).Decode(&rawChecks); err != nil {
		zap.L().Fatal("Cannot decode yaml structure.", zap.Error(err))
	}

	hc := &HealthChecker{
		notifier: notifier,
	}
	hc.checks = hc.mapRawChecks(rawChecks, "")

	return hc
}

func (hc *HealthChecker) mapRawChecks(rawChecks []rawCheck, parentId string) []Check {
	var checks []Check
	for _, rawCheck := range rawChecks {
		baseCheck := &BaseCheck{
			Name:     rawCheck.StringValue("name"),
			Type:     rawCheck.StringValue("type"),
			notifier: hc.notifier,
		}
		baseCheck.id = buildId(baseCheck.Name)
		if parentId != "" {
			baseCheck.id = parentId + "." + baseCheck.id
		}

		var check Check

		switch baseCheck.Type {
		case "group":
			check = &Group{
				BaseCheck: baseCheck,
				Checks:    hc.mapRawChecks(rawCheck.SubChecks("checks"), baseCheck.id),
			}
		case "tls":
			check = &TlsCheck{
				BaseCheck: baseCheck,
				Address:   rawCheck.StringValue("address"),
				Insecure:  rawCheck.BoolValue("insecure"),
			}
		case "smtp":
			check = &SmtpCheck{
				BaseCheck: baseCheck,
				Address:   rawCheck.StringValue("address"),
			}
		default:
			zap.L().Fatal("Unrecognized type.", zap.String("type", baseCheck.Type))
		}

		checks = append(checks, check)
	}

	return checks
}

func (hc *HealthChecker) PerformChecks() {
	zap.L().Info("Running checks.")
	for _, c := range hc.checks {
		c.Perform()
	}
}

type rawCheck map[interface{}]interface{}

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

var nonWordChars = regexp.MustCompile(`[^\w]`)
var separatorChain = regexp.MustCompile(`_+`)

func buildId(s string) string {
	s = strings.ToLower(s)
	s = nonWordChars.ReplaceAllString(s, "_")
	s = separatorChain.ReplaceAllString(s, "_")
	return s
}
