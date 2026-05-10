package main

import (
	"fmt"
	"slices"
	"strings"
)

type Values struct {
	Domain        string `yaml:"domain"`
	StorageClass  string `yaml:"storageClass"`
	StorageSize   string `yaml:"storageSize"`
	AdminUsername string `yaml:"adminUsername"`
	AdminPassword string `yaml:"adminPassword"`
}

var defaults = Values{
	Domain:        "forgejo.local",
	StorageClass:  "local-path",
	StorageSize:   "10Gi",
	AdminUsername: "gitadmin",
}

// Reserved by Forgejo — sourced from models/user/user.go reservedUsernames.
var reservedUsernames = []string{
	".", "..", "-", ".well-known",
	"api", "metrics", "v2",
	"assets", "attachments",
	"avatar", "avatars", "repo-avatars",
	"captcha", "login", "org", "repo", "user",
	"admin", "explore", "issues", "pulls", "milestones",
	"notifications", "report_abuse",
}

func (v Values) validate() error {
	if strings.Contains(v.Domain, ":") {
		return fmt.Errorf("domain %q must not include a port", v.Domain)
	}
	if slices.Contains(reservedUsernames, v.AdminUsername) {
		return fmt.Errorf("adminUsername %q is reserved by Forgejo", v.AdminUsername)
	}
	return nil
}
