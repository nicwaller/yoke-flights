package main

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"text/template"

	corev1 "k8s.io/api/core/v1"

	"github.com/yokecd/yoke/pkg/flight/wasi/k8s"
)

//go:embed app.ini.tmpl
var templateFS embed.FS

var appIniTmpl = template.Must(template.ParseFS(templateFS, "app.ini.tmpl"))

// resolveAppIni returns the existing app.ini from the cluster if present,
// otherwise generates a fresh one with new random secrets.
func resolveAppIni(name, ns, domain string) (string, error) {
	existing, err := lookupSecret(name+"-config", ns)
	if err != nil {
		return "", fmt.Errorf("failed to lookup config secret: %w", err)
	}
	if existing != nil {
		if ini := string(existing.Data["app.ini"]); ini != "" {
			return ini, nil
		}
	}
	return generateAppIni(domain)
}

// resolveAdminPassword returns the provided password, or the existing one from
// the cluster, or a newly generated one — in that order of preference.
func resolveAdminPassword(name, ns, provided string) (string, error) {
	if provided != "" {
		return provided, nil
	}
	existing, err := lookupSecret(name+"-admin", ns)
	if err != nil {
		return "", fmt.Errorf("failed to lookup admin secret: %w", err)
	}
	if existing != nil {
		return string(existing.Data["password"]), nil
	}
	return randomSecret(16)
}

func lookupSecret(name, ns string) (*corev1.Secret, error) {
	s, err := k8s.Lookup[corev1.Secret](k8s.ResourceIdentifier{
		ApiVersion: "v1",
		Kind:       "Secret",
		Name:       name,
		Namespace:  ns,
	})
	if err != nil && !k8s.IsErrNotFound(err) && !errors.Is(err, k8s.ErrorClusterAccessNotGranted) {
		return nil, err
	}
	return s, nil
}

func generateAppIni(domain string) (string, error) {
	lfsSecret, err := randomSecret(32)
	if err != nil {
		return "", err
	}
	internalToken, err := randomSecret(32)
	if err != nil {
		return "", err
	}
	secretKey, err := randomSecret(32)
	if err != nil {
		return "", err
	}
	jwtSecret, err := randomSecret(32)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := appIniTmpl.Execute(&buf, struct {
		Domain, LFSJWTSecret, InternalToken, SecretKey, JWTSecret string
		HTTPPort, SSHPort                                          int
	}{
		Domain:        domain,
		LFSJWTSecret:  lfsSecret,
		InternalToken: internalToken,
		SecretKey:     secretKey,
		JWTSecret:     jwtSecret,
		HTTPPort:      httpPort,
		SSHPort:       sshExternalPort,
	}); err != nil {
		return "", fmt.Errorf("failed to render app.ini: %w", err)
	}
	return buf.String(), nil
}
