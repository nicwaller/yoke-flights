package main

import "testing"

func TestValidate(t *testing.T) {
	for _, svcType := range []string{"ClusterIP", "NodePort", "LoadBalancer", "ExternalName"} {
		v := defaults
		v.HTTPServiceType = svcType
		v.SSHServiceType = svcType
		if err := v.validate(); err != nil {
			t.Errorf("unexpected error for valid type %q: %v", svcType, err)
		}
	}
}

func TestValidateInvalidHTTPServiceType(t *testing.T) {
	v := defaults
	v.HTTPServiceType = "BadType"
	if err := v.validate(); err == nil {
		t.Fatal("expected error for invalid httpServiceType")
	}
}

func TestValidateInvalidSSHServiceType(t *testing.T) {
	v := defaults
	v.SSHServiceType = "BadType"
	if err := v.validate(); err == nil {
		t.Fatal("expected error for invalid sshServiceType")
	}
}

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
