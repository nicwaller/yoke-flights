package main

import (
	"crypto/rand"
	"math/big"
	"strings"

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

const secretCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*_+-"

func randomSecret(length int) (string, error) {
	n := big.NewInt(int64(len(secretCharset)))
	var sb strings.Builder
	for range length {
		i, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", err
		}
		sb.WriteByte(secretCharset[i.Int64()])
	}
	return sb.String(), nil
}

func ptr[T any](v T) *T { return &v }
