package main

import (
	"crypto/rand"
	"encoding/base64"

	corev1 "k8s.io/api/core/v1"
)

func envBase() []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "GITEA_APP_INI", Value: "/data/gitea/conf/app.ini"},
		{Name: "GITEA_CUSTOM", Value: "/data/gitea"},
		{Name: "GITEA_WORK_DIR", Value: "/data"},
		{Name: "GITEA_TEMP", Value: "/tmp/gitea"},
		{Name: "HOME", Value: "/data/gitea/git"},
	}
}

func randomSecret(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func ptr[T any](v T) *T { return &v }
