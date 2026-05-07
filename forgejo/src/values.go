package main

import (
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"
)

type Values struct {
	Domain          string `yaml:"domain"`
	StorageClass    string `yaml:"storageClass"`
	StorageSize     string `yaml:"storageSize"`
	AdminUsername   string `yaml:"adminUsername"`
	AdminPassword   string `yaml:"adminPassword"`
	HTTPServiceType string `yaml:"httpServiceType"`
	SSHServiceType  string `yaml:"sshServiceType"`
}

var defaults = Values{
	Domain:          "forgejo.local",
	StorageClass:    "local-path",
	StorageSize:     "10Gi",
	AdminUsername:   "gitadmin",
	HTTPServiceType: "LoadBalancer",
	SSHServiceType:  "LoadBalancer",
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

var validServiceTypes = map[corev1.ServiceType]bool{
	corev1.ServiceTypeClusterIP:    true,
	corev1.ServiceTypeNodePort:     true,
	corev1.ServiceTypeLoadBalancer: true,
	corev1.ServiceTypeExternalName: true,
}

func (v Values) validate() error {
	if slices.Contains(reservedUsernames, v.AdminUsername) {
		return fmt.Errorf("adminUsername %q is reserved by Forgejo", v.AdminUsername)
	}
	if !validServiceTypes[corev1.ServiceType(v.HTTPServiceType)] {
		return fmt.Errorf("invalid httpServiceType %q", v.HTTPServiceType)
	}
	if !validServiceTypes[corev1.ServiceType(v.SSHServiceType)] {
		return fmt.Errorf("invalid sshServiceType %q", v.SSHServiceType)
	}
	return nil
}
