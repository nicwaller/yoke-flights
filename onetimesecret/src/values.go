package main

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

type Values struct {
	Domain            string `yaml:"domain"`
	Port              int    `yaml:"port"`
	ServiceType       string `yaml:"serviceType"`
	Colonel           string `yaml:"colonel"`
	SSL               bool   `yaml:"ssl"`
	SmtpHost          string `yaml:"smtpHost"`
	SmtpPort          int    `yaml:"smtpPort"`
	SmtpUsername      string `yaml:"smtpUsername"`
	SmtpPassword      string `yaml:"smtpPassword"`
	FromEmail         string `yaml:"fromEmail"`
	AuthSignup        bool   `yaml:"authSignup"`
	AuthSignin        bool   `yaml:"authSignin"`
	RedisStorageClass string `yaml:"redisStorageClass"`
	RedisStorageSize  string `yaml:"redisStorageSize"`
}

var validServiceTypes = map[corev1.ServiceType]bool{
	corev1.ServiceTypeClusterIP:    true,
	corev1.ServiceTypeNodePort:     true,
	corev1.ServiceTypeLoadBalancer: true,
	corev1.ServiceTypeExternalName: true,
}

var defaults = Values{
	Domain:      "onetimesecret.local",
	Port:        3001,
	ServiceType: "LoadBalancer",
	Colonel:     "admin@onetimesecret.local",
	SSL:               false,
	SmtpPort:          587,
	FromEmail:         "no-reply@onetimesecret.local",
	AuthSignup:        true,
	AuthSignin:        true,
	RedisStorageClass: "local-path",
	RedisStorageSize:  "1Gi",
}

func (v Values) validate() error {
	if strings.Contains(v.Domain, ":") {
		return fmt.Errorf("domain %q must not include a port", v.Domain)
	}
	if !validServiceTypes[corev1.ServiceType(v.ServiceType)] {
		return fmt.Errorf("invalid serviceType %q", v.ServiceType)
	}
	if v.Port < 1 || v.Port > 65535 {
		return fmt.Errorf("port %d is out of range", v.Port)
	}
	if v.SmtpPort < 1 || v.SmtpPort > 65535 {
		return fmt.Errorf("smtpPort %d is out of range", v.SmtpPort)
	}
	return nil
}
