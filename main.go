package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
)

type Check interface{}

type BaseCheck struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

type Group struct {
	BaseCheck
	Checks []Check
}

type TlsCheck struct {
	BaseCheck
	Address  string `yaml:"address"`
	Insecure bool   `yaml:"insecure"`
}

type rawCheck map[interface{}]interface{}

func main() {
	initLogger()

	f, err := os.Open("checks.yaml")
	if err != nil {
		zap.L().Fatal("Cannot open check definitions.", zap.Error(err))
	}

	var rawChecks []rawCheck
	if err := yaml.NewDecoder(f).Decode(&rawChecks); err != nil {
		zap.L().Fatal("Cannot decode yaml structure.", zap.Error(err))
	}

	checks := mapRawChecks(rawChecks)
	fmt.Println(checks)
}

func mapRawChecks(rawChecks []rawCheck) []Check {
	var checks []Check
	for _, rawCheck := range rawChecks {
		baseCheck := BaseCheck{
			Name: rawCheck.StringValue("name"),
			Type: rawCheck.StringValue("type"),
		}

		var check Check

		switch baseCheck.Type {
		case "group":
			check = Group{
				BaseCheck: baseCheck,
				Checks:    mapRawChecks(rawCheck.SubChecks("checks")),
			}
		case "tls":
			check = TlsCheck{
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
