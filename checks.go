package main

import (
	"crypto/tls"
	"net"
	"net/smtp"
	"time"

	"go.uber.org/zap"
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
	b.lastCheckTime = time.Now()
	if ok != b.lastOk || b.lastStateChange.IsZero() {
		b.lastStateChange = time.Now()
		b.lastOk = ok
		state := CheckState{
			Timestamp: b.lastCheckTime,
			Ok:        ok,
			Reason:    reason,
		}
		if err := b.notifier.Notify(b.Name, state); err != nil {
			zap.L().Error("Cannot notify.", zap.String("name", b.Name), zap.Error(err))
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
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}
	conn, err := tls.DialWithDialer(dialer, "tcp", t.Address, &tls.Config{
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

type SmtpCheck struct {
	*BaseCheck
	Address string `yaml:"address"`
}

func (s *SmtpCheck) Perform() bool {
	c, err := smtp.Dial(s.Address)
	if err != nil {
		s.UpdateState(false, err.Error())
		return false
	} else {
		s.UpdateState(true, "")
		c.Close()
		return true
	}
}
