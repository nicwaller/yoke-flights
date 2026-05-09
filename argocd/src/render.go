package main

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

const (
	argocdImage = "quay.io/argoproj/argocd:v3.4.1"
	redisImage  = "public.ecr.aws/docker/library/redis:8.2.3-alpine"
)

func render(_, ns string, _ Values) ([]json.RawMessage, error) {
	crds, err := parseCRDs()
	if err != nil {
		return nil, err
	}

	objects := []any{
		buildSA("argocd-application-controller", ns),
		buildSA("argocd-applicationset-controller", ns),
		buildSA("argocd-redis", ns),
		buildSA("argocd-repo-server", ns),

		buildRoleApplicationController(ns),
		buildRoleApplicationsetController(ns),
		buildRoleRedis(ns),

		buildClusterRoleApplicationController(),

		buildRoleBinding("argocd-application-controller", ns),
		buildRoleBinding("argocd-applicationset-controller", ns),
		buildRoleBinding("argocd-redis", ns),

		buildClusterRoleBindingApplicationController(ns),

		buildConfigMapArgocdCm(ns),
		buildConfigMapCmdParams(ns),
		buildConfigMapGpgKeys(ns),
		buildConfigMapRbacCm(ns),
		buildConfigMapSshKnownHosts(ns),
		buildConfigMapTlsCerts(ns),

		buildSecretArgocd(ns),

		buildServiceApplicationsetController(ns),
		buildServiceMetrics(ns),
		buildServiceRedis(ns),
		buildServiceRepoServer(ns),

		buildDeploymentApplicationsetController(ns),
		buildDeploymentRedis(ns),
		buildDeploymentRepoServer(ns),
		buildStatefulSetApplicationController(ns),
	}

	result := make([]json.RawMessage, len(crds), len(crds)+len(objects))
	copy(result, crds)
	for _, obj := range objects {
		b, err := json.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource: %w", err)
		}
		result = append(result, b)
	}
	return result, nil
}

func argoLabels(component, name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/component": component,
		"app.kubernetes.io/name":      name,
		"app.kubernetes.io/part-of":   "argocd",
	}
}

func cmRef(envName, cm, key string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: envName,
		ValueFrom: &corev1.EnvVarSource{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: cm},
				Key:                  key,
				Optional:             ptr(true),
			},
		},
	}
}

func secretRef(envName, secret, key string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: envName,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: secret},
				Key:                  key,
			},
		},
	}
}

func restrictedSecCtx() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr(false),
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		ReadOnlyRootFilesystem:   ptr(true),
		RunAsNonRoot:             ptr(true),
		SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
	}
}
