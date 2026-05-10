package main

import "testing"

func TestValidateDomainWithPort(t *testing.T) {
	v := defaults
	v.Domain = "forgejo.local:3000"
	if err := v.validate(); err == nil {
		t.Fatal("expected error for domain containing a port")
	}
}

func TestValidateReservedAdminUsername(t *testing.T) {
	for _, name := range []string{"admin", "api", "explore", "user", "org"} {
		v := defaults
		v.AdminUsername = name
		if err := v.validate(); err == nil {
			t.Errorf("expected error for reserved username %q", name)
		}
	}
}

func TestValidateValidAdminUsername(t *testing.T) {
	v := defaults
	v.AdminUsername = "gitadmin"
	if err := v.validate(); err != nil {
		t.Errorf("unexpected error for valid username: %v", err)
	}
}
