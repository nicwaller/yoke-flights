package main

import (
	"strings"
	"testing"
)

func TestGenerateAppIni(t *testing.T) {
	const domain = "git.example.com"
	ini, err := generateAppIni(domain)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"DOMAIN = " + domain,
		"SSH_DOMAIN = " + domain,
		"ROOT_URL = https://" + domain + "/",
		"LFS_JWT_SECRET =",
		"INTERNAL_TOKEN =",
		"SECRET_KEY =",
		"JWT_SECRET =",
		"INSTALL_LOCK = true",
	} {
		if !strings.Contains(ini, want) {
			t.Errorf("app.ini missing %q", want)
		}
	}
	if strings.Contains(ini, "{{") {
		t.Error("app.ini contains unrendered template markers")
	}
}
