package main

import (
	"testing"
)

func TestValidate_DomainWithPort(t *testing.T) {
	v := defaults
	v.Domain = "onetimesecret.local:3000"
	if err := v.validate(); err == nil {
		t.Error("expected error for domain with port")
	}
}

func TestValidate_InvalidSmtpPort(t *testing.T) {
	v := defaults
	v.SmtpPort = 0
	if err := v.validate(); err == nil {
		t.Error("expected error for smtpPort 0")
	}
}

func TestValidate_DefaultsAreValid(t *testing.T) {
	if err := defaults.validate(); err != nil {
		t.Errorf("defaults should be valid: %v", err)
	}
}
