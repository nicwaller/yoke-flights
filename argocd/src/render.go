package main

import (
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

// --- helpers ---

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

// --- ServiceAccounts ---

func buildSA(name, ns string) corev1.ServiceAccount {
	return corev1.ServiceAccount{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
	}
}

// --- Roles ---

func buildRoleApplicationController(ns string) rbacv1.Role {
	return rbacv1.Role{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "Role"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-application-controller", Namespace: ns},
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{""}, Resources: []string{"secrets", "configmaps"}, Verbs: []string{"get", "list", "watch"}},
			{APIGroups: []string{"argoproj.io"}, Resources: []string{"applications", "applicationsets", "appprojects"}, Verbs: []string{"create", "get", "list", "watch", "update", "patch", "delete"}},
			{APIGroups: []string{""}, Resources: []string{"events"}, Verbs: []string{"create", "list"}},
			{APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"get", "list", "watch"}},
		},
	}
}

func buildRoleApplicationsetController(ns string) rbacv1.Role {
	return rbacv1.Role{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "Role"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-applicationset-controller", Namespace: ns},
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{"argoproj.io"}, Resources: []string{"applications", "applicationsets", "applicationsets/finalizers"}, Verbs: []string{"create", "delete", "get", "list", "patch", "update", "watch"}},
			{APIGroups: []string{"argoproj.io"}, Resources: []string{"appprojects"}, Verbs: []string{"get", "list", "watch"}},
			{APIGroups: []string{"argoproj.io"}, Resources: []string{"applicationsets/status"}, Verbs: []string{"get", "patch", "update"}},
			{APIGroups: []string{""}, Resources: []string{"events"}, Verbs: []string{"create", "get", "list", "patch", "watch"}},
			{APIGroups: []string{""}, Resources: []string{"secrets", "configmaps"}, Verbs: []string{"get", "list", "watch"}},
			{APIGroups: []string{"coordination.k8s.io"}, Resources: []string{"leases"}, Verbs: []string{"create"}},
			{APIGroups: []string{"coordination.k8s.io"}, ResourceNames: []string{"58ac56fa.applicationsets.argoproj.io"}, Resources: []string{"leases"}, Verbs: []string{"get", "update", "create"}},
		},
	}
}

func buildRoleRedis(ns string) rbacv1.Role {
	return rbacv1.Role{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "Role"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-redis", Namespace: ns},
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{""}, ResourceNames: []string{"argocd-redis"}, Resources: []string{"secrets"}, Verbs: []string{"get"}},
			{APIGroups: []string{""}, Resources: []string{"secrets"}, Verbs: []string{"create"}},
		},
	}
}

func buildClusterRoleApplicationController() rbacv1.ClusterRole {
	return rbacv1.ClusterRole{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "ClusterRole"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-application-controller"},
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}},
			{NonResourceURLs: []string{"*"}, Verbs: []string{"*"}},
		},
	}
}

// --- RoleBindings ---

func buildRoleBinding(name, ns string) rbacv1.RoleBinding {
	return rbacv1.RoleBinding{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "RoleBinding"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "Role", Name: name},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: name, Namespace: ns}},
	}
}

func buildClusterRoleBindingApplicationController(ns string) rbacv1.ClusterRoleBinding {
	return rbacv1.ClusterRoleBinding{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "ClusterRoleBinding"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-application-controller"},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "argocd-application-controller"},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: "argocd-application-controller", Namespace: ns}},
	}
}

// --- ConfigMaps ---

func buildConfigMapArgocdCm(ns string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-cm", Namespace: ns},
		Data: map[string]string{
			"resource.exclusions": `### Network resources created by the Kubernetes control plane and excluded to reduce the number of watched events and UI clutter
- apiGroups:
  - ''
  - discovery.k8s.io
  kinds:
  - Endpoints
  - EndpointSlice
### Internal Kubernetes resources excluded reduce the number of watched events
- apiGroups:
  - coordination.k8s.io
  kinds:
  - Lease
### Internal Kubernetes Authz/Authn resources excluded reduce the number of watched events
- apiGroups:
  - authentication.k8s.io
  - authorization.k8s.io
  kinds:
  - SelfSubjectReview
  - TokenReview
  - LocalSubjectAccessReview
  - SelfSubjectAccessReview
  - SelfSubjectRulesReview
  - SubjectAccessReview
### Intermediate Certificate Request excluded reduce the number of watched events
- apiGroups:
  - certificates.k8s.io
  kinds:
  - CertificateSigningRequest
- apiGroups:
  - cert-manager.io
  kinds:
  - CertificateRequest
### Cilium internal resources excluded reduce the number of watched events and UI Clutter
- apiGroups:
  - cilium.io
  kinds:
  - CiliumIdentity
  - CiliumEndpoint
  - CiliumEndpointSlice
### Kyverno intermediate and reporting resources excluded reduce the number of watched events and improve performance
- apiGroups:
  - kyverno.io
  - reports.kyverno.io
  - wgpolicyk8s.io
  kinds:
  - PolicyReport
  - ClusterPolicyReport
  - EphemeralReport
  - ClusterEphemeralReport
  - AdmissionReport
  - ClusterAdmissionReport
  - BackgroundScanReport
  - ClusterBackgroundScanReport
  - UpdateRequest
`,
			"resource.customizations.ignoreResourceUpdates.all": `jsonPointers:
  - /status
`,
			"resource.customizations.ignoreResourceUpdates.ConfigMap": `jqPathExpressions:
  # Ignore the cluster-autoscaler status
  - '.metadata.annotations."cluster-autoscaler.kubernetes.io/last-updated"'
  # Ignore the annotation of the legacy Leases election
  - '.metadata.annotations."control-plane.alpha.kubernetes.io/leader"'
`,
			"resource.customizations.ignoreResourceUpdates.Endpoints": `jsonPointers:
  - /metadata
  - /subsets
`,
			"resource.customizations.ignoreResourceUpdates.apps_ReplicaSet": `jqPathExpressions:
  - '.metadata.annotations."deployment.kubernetes.io/desired-replicas"'
  - '.metadata.annotations."deployment.kubernetes.io/max-replicas"'
  - '.metadata.annotations."rollout.argoproj.io/desired-replicas"'
`,
			"resource.customizations.ignoreResourceUpdates.argoproj.io_Application": `jqPathExpressions:
  - '.metadata.annotations."notified.notifications.argoproj.io"'
  - '.metadata.annotations."argocd.argoproj.io/refresh"'
  - '.metadata.annotations."argocd.argoproj.io/hydrate"'
  - '.operation'
`,
			"resource.customizations.ignoreResourceUpdates.argoproj.io_Rollout": `jqPathExpressions:
  - '.metadata.annotations."notified.notifications.argoproj.io"'
`,
			"resource.customizations.ignoreResourceUpdates.autoscaling_HorizontalPodAutoscaler": `jqPathExpressions:
  - '.metadata.annotations."autoscaling.alpha.kubernetes.io/behavior"'
  - '.metadata.annotations."autoscaling.alpha.kubernetes.io/conditions"'
  - '.metadata.annotations."autoscaling.alpha.kubernetes.io/metrics"'
  - '.metadata.annotations."autoscaling.alpha.kubernetes.io/current-metrics"'
`,
			"resource.customizations.ignoreResourceUpdates.discovery.k8s.io_EndpointSlice": `jsonPointers:
  - /metadata
  - /endpoints
  - /ports
`,
		},
	}
}

func buildConfigMapCmdParams(ns string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-cmd-params-cm", Namespace: ns},
	}
}

func buildConfigMapGpgKeys(ns string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-gpg-keys-cm", Namespace: ns},
	}
}

func buildConfigMapRbacCm(ns string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-rbac-cm", Namespace: ns},
	}
}

func buildConfigMapSshKnownHosts(ns string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-ssh-known-hosts-cm", Namespace: ns},
		Data: map[string]string{
			"ssh_known_hosts": `# This file was automatically generated by hack/update-ssh-known-hosts.sh. DO NOT EDIT
[ssh.github.com]:443 ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg=
[ssh.github.com]:443 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl
[ssh.github.com]:443 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQowgcQnjshcLrqPEiiphnt+VTTvDP6mHBL9j1aNUkY4Ue1gvwnGLVlOhGeYrnZaMgRK6+PKCUXaDbC7qtbW8gIkhL7aGCsOr/C56SJMy/BCZfxd1nWzAOxSDPgVsmerOBYfNqltV9/hWCqBywINIR+5dIg6JTJ72pcEpEjcYgXkE2YEFXV1JHnsKgbLWNlhScqb2UmyRkQyytRLtL+38TGxkxCflmO+5Z8CSSNY7GidjMIZ7Q4zMjA2n1nGrlTDkzwDCsw+wqFPGQA179cnfGWOWRVruj16z6XyvxvjJwbz0wQZ75XK5tKSb7FNyeIEs4TT4jk+S4dhPeAUC5y+bDYirYgM4GC7uEnztnZyaVWQ7B381AK4Qdrwt51ZqExKbQpTUNn+EjqoTwvqNj4kqx5QUCI0ThS/YkOxJCXmPUWZbhjpCg56i+2aB6CmK2JGhn57K5mj0MNdBXA4/WnwH6XoPWJzK5Nyu2zB3nAZp+S5hpQs+p1vN1/wsjk=
bitbucket.org ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBPIQmuzMBuKdWeF4+a2sjSSpBK0iqitSQ+5BM9KhpexuGt20JpTVM7u5BDZngncgrqDMbWdxMWWOGtZ9UgbqgZE=
bitbucket.org ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIazEu89wgQZ4bqs3d63QSMzYVa0MuJ2e2gKTKqu+UUO
bitbucket.org ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDQeJzhupRu0u0cdegZIa8e86EG2qOCsIsD1Xw0xSeiPDlCr7kq97NLmMbpKTX6Esc30NuoqEEHCuc7yWtwp8dI76EEEB1VqY9QJq6vk+aySyboD5QF61I/1WeTwu+deCbgKMGbUijeXhtfbxSxm6JwGrXrhBdofTsbKRUsrN1WoNgUa8uqN1Vx6WAJw1JHPhglEGGHea6QICwJOAr/6mrui/oB7pkaWKHj3z7d1IC4KWLtY47elvjbaTlkN04Kc/5LFEirorGYVbt15kAUlqGM65pk6ZBxtaO3+30LVlORZkxOh+LKL/BvbZ/iRNhItLqNyieoQj/uh/7Iv4uyH/cV/0b4WDSd3DptigWq84lJubb9t/DnZlrJazxyDCulTmKdOR7vs9gMTo+uoIrPSb8ScTtvw65+odKAlBj59dhnVp9zd7QUojOpXlL62Aw56U4oO+FALuevvMjiWeavKhJqlR7i5n9srYcrNV7ttmDw7kf/97P5zauIhxcjX+xHv4M=
github.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg=
github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl
github.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQowgcQnjshcLrqPEiiphnt+VTTvDP6mHBL9j1aNUkY4Ue1gvwnGLVlOhGeYrnZaMgRK6+PKCUXaDbC7qtbW8gIkhL7aGCsOr/C56SJMy/BCZfxd1nWzAOxSDPgVsmerOBYfNqltV9/hWCqBywINIR+5dIg6JTJ72pcEpEjcYgXkE2YEFXV1JHnsKgbLWNlhScqb2UmyRkQyytRLtL+38TGxkxCflmO+5Z8CSSNY7GidjMIZ7Q4zMjA2n1nGrlTDkzwDCsw+wqFPGQA179cnfGWOWRVruj16z6XyvxvjJwbz0wQZ75XK5tKSb7FNyeIEs4TT4jk+S4dhPeAUC5y+bDYirYgM4GC7uEnztnZyaVWQ7B381AK4Qdrwt51ZqExKbQpTUNn+EjqoTwvqNj4kqx5QUCI0ThS/YkOxJCXmPUWZbhjpCg56i+2aB6CmK2JGhn57K5mj0MNdBXA4/WnwH6XoPWJzK5Nyu2zB3nAZp+S5hpQs+p1vN1/wsjk=
gitlab.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBFSMqzJeV9rUzU4kWitGjeR4PWSa29SPqJ1fVkhtj3Hw9xjLVXVYrU9QlYWrOLXBpQ6KWjbjTDTdDkoohFzgbEY=
gitlab.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAfuCHKVTjquxvt6CM6tdG4SLp1Btn/nOeHHE5UOzRdf
gitlab.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCsj2bNKTBSpIYDEGk9KxsGh3mySTRgMtXL583qmBpzeQ+jqCMRgBqB98u3z++J1sKlXHWfM9dyhSevkMwSbhoR8XIq/U0tCNyokEi/ueaBMCvbcTHhO7FcwzY92WK4Yt0aGROY5qX2UKSeOvuP4D6TPqKF1onrSzH9bx9XUf2lEdWT/ia1NEKjunUqu1xOB/StKDHMoX4/OKyIzuS0q/T1zOATthvasJFoPrAjkohTyaDUz2LN5JoH839hViyEG82yB+MjcFV5MU3N1l1QL3cVUCh93xSaua1N85qivl+siMkPGbO5xR/En4iEY6K2XPASUEMaieWVNTRCtJ4S8H+9
ssh.dev.azure.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7Hr1oTWqNqOlzGJOfGJ4NakVyIzf1rXYd4d7wo6jBlkLvCA4odBlL0mDUyZ0/QUfTTqeu+tm22gOsv+VrVTMk6vwRU75gY/y9ut5Mb3bR5BV58dKXyq9A9UeB5Cakehn5Zgm6x1mKoVyf+FFn26iYqXJRgzIZZcZ5V6hrE0Qg39kZm4az48o0AUbf6Sp4SLdvnuMa2sVNwHBboS7EJkm57XQPVU3/QpyNLHbWDdzwtrlS+ez30S3AdYhLKEOxAG8weOnyrtLJAUen9mTkol8oII1edf7mWWbWVf0nBmly21+nZcmCTISQBtdcyPaEno7fFQMDD26/s0lfKob4Kw8H
vs-ssh.visualstudio.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7Hr1oTWqNqOlzGJOfGJ4NakVyIzf1rXYd4d7wo6jBlkLvCA4odBlL0mDUyZ0/QUfTTqeu+tm22gOsv+VrVTMk6vwRU75gY/y9ut5Mb3bR5BV58dKXyq9A9UeB5Cakehn5Zgm6x1mKoVyf+FFn26iYqXJRgzIZZcZ5V6hrE0Qg39kZm4az48o0AUbf6Sp4SLdvnuMa2sVNwHBboS7EJkm57XQPVU3/QpyNLHbWDdzwtrlS+ez30S3AdYhLKEOxAG8weOnyrtLJAUen9mTkol8oII1edf7mWWbWVf0nBmly21+nZcmCTISQBtdcyPaEno7fFQMDD26/s0lfKob4Kw8H
`,
		},
	}
}

func buildConfigMapTlsCerts(ns string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-tls-certs-cm", Namespace: ns},
	}
}

// --- Secrets ---

func buildSecretArgocd(ns string) corev1.Secret {
	return corev1.Secret{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-secret", Namespace: ns},
		Type:       corev1.SecretTypeOpaque,
	}
}

// --- Services ---

func buildServiceApplicationsetController(ns string) corev1.Service {
	labels := argoLabels("applicationset-controller", "argocd-applicationset-controller")
	return corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-applicationset-controller", Namespace: ns, Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app.kubernetes.io/name": "argocd-applicationset-controller"},
			Ports: []corev1.ServicePort{
				{Name: "webhook", Port: 7000, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromString("webhook")},
				{Name: "metrics", Port: 8080, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromString("metrics")},
			},
		},
	}
}

func buildServiceMetrics(ns string) corev1.Service {
	labels := argoLabels("metrics", "argocd-metrics")
	return corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-metrics", Namespace: ns, Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app.kubernetes.io/name": "argocd-application-controller"},
			Ports: []corev1.ServicePort{
				{Name: "metrics", Port: 8082, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt32(8082)},
			},
		},
	}
}

func buildServiceRedis(ns string) corev1.Service {
	labels := argoLabels("redis", "argocd-redis")
	return corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-redis", Namespace: ns, Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app.kubernetes.io/name": "argocd-redis"},
			Ports: []corev1.ServicePort{
				{Name: "tcp-redis", Port: 6379, TargetPort: intstr.FromInt32(6379)},
			},
		},
	}
}

func buildServiceRepoServer(ns string) corev1.Service {
	labels := argoLabels("repo-server", "argocd-repo-server")
	return corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-repo-server", Namespace: ns, Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app.kubernetes.io/name": "argocd-repo-server"},
			Ports: []corev1.ServicePort{
				{Name: "server", Port: 8081, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt32(8081)},
				{Name: "metrics", Port: 8084, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt32(8084)},
			},
		},
	}
}

// --- Deployments ---

func buildDeploymentApplicationsetController(ns string) appsv1.Deployment {
	labels := argoLabels("applicationset-controller", "argocd-applicationset-controller")
	podLabels := map[string]string{"app.kubernetes.io/name": "argocd-applicationset-controller"}
	return appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-applicationset-controller", Namespace: ns, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: podLabels},
				Spec: corev1.PodSpec{
					ServiceAccountName: "argocd-applicationset-controller",
					NodeSelector:       map[string]string{"kubernetes.io/os": "linux"},
					Containers: []corev1.Container{
						{
							Name:            "argocd-applicationset-controller",
							Image:           argocdImage,
							ImagePullPolicy: corev1.PullAlways,
							Args:            []string{"/usr/local/bin/argocd-applicationset-controller"},
							Ports: []corev1.ContainerPort{
								{Name: "webhook", ContainerPort: 7000},
								{Name: "metrics", ContainerPort: 8080},
							},
							Env: []corev1.EnvVar{
								cmRef("NAMESPACE", "argocd-cmd-params-cm", "applicationsetcontroller.namespace"),
								cmRef("ARGOCD_APPLICATIONSET_CONTROLLER_GLOBAL_PRESERVED_ANNOTATIONS", "argocd-cmd-params-cm", "applicationsetcontroller.global.preserved.annotations"),
								cmRef("ARGOCD_APPLICATIONSET_CONTROLLER_GLOBAL_PRESERVED_LABELS", "argocd-cmd-params-cm", "applicationsetcontroller.global.preserved.labels"),
								cmRef("ARGOCD_APPLICATIONSET_CONTROLLER_ENABLE_LEADER_ELECTION", "argocd-cmd-params-cm", "applicationsetcontroller.enable.leader.election"),
								cmRef("ARGOCD_APPLICATIONSET_CONTROLLER_REPO_SERVER", "argocd-cmd-params-cm", "repo.server"),
								cmRef("ARGOCD_APPLICATIONSET_CONTROLLER_POLICY", "argocd-cmd-params-cm", "applicationsetcontroller.policy"),
								cmRef("ARGOCD_APPLICATIONSET_CONTROLLER_DEBUG", "argocd-cmd-params-cm", "applicationsetcontroller.debug"),
								cmRef("ARGOCD_APPLICATIONSET_CONTROLLER_LOGFORMAT", "argocd-cmd-params-cm", "applicationsetcontroller.log.format"),
								cmRef("ARGOCD_APPLICATIONSET_CONTROLLER_LOGLEVEL", "argocd-cmd-params-cm", "applicationsetcontroller.log.level"),
								cmRef("ARGOCD_APPLICATIONSET_CONTROLLER_ENABLE_PROGRESSIVE_SYNCS", "argocd-cmd-params-cm", "applicationsetcontroller.enable.progressive.syncs"),
								cmRef("ARGOCD_APPLICATIONSET_CONTROLLER_REPO_SERVER_PLAINTEXT", "argocd-cmd-params-cm", "applicationsetcontroller.repo.server.plaintext"),
								cmRef("ARGOCD_APPLICATIONSET_CONTROLLER_NAMESPACES", "argocd-cmd-params-cm", "applicationsetcontroller.namespaces"),
								cmRef("ARGOCD_APPLICATIONSET_CONTROLLER_ENABLE_SCM_PROVIDERS", "argocd-cmd-params-cm", "applicationsetcontroller.enable.scm.providers"),
								cmRef("GRPC_ENABLE_TXT_SERVICE_CONFIG", "argocd-cmd-params-cm", "applicationsetcontroller.grpc.enable.txt.service.config"),
							},
							SecurityContext: restrictedSecCtx(),
							VolumeMounts: []corev1.VolumeMount{
								{Name: "ssh-known-hosts", MountPath: "/app/config/ssh"},
								{Name: "tls-certs", MountPath: "/app/config/tls"},
								{Name: "gpg-keys", MountPath: "/app/config/gpg/source"},
								{Name: "gpg-keyring", MountPath: "/app/config/gpg/keys"},
								{Name: "tmp", MountPath: "/tmp"},
								{Name: "argocd-repo-server-tls", MountPath: "/app/config/reposerver/tls"},
								{Name: "argocd-cmd-params-cm", MountPath: "/home/argocd/params"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{Name: "ssh-known-hosts", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "argocd-ssh-known-hosts-cm"}}}},
						{Name: "tls-certs", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "argocd-tls-certs-cm"}}}},
						{Name: "gpg-keys", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "argocd-gpg-keys-cm"}}}},
						{Name: "gpg-keyring", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "tmp", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "argocd-repo-server-tls", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
							SecretName: "argocd-repo-server-tls",
							Optional:   ptr(true),
							Items: []corev1.KeyToPath{
								{Key: "tls.crt", Path: "tls.crt"},
								{Key: "tls.key", Path: "tls.key"},
								{Key: "ca.crt", Path: "ca.crt"},
							},
						}}},
						{Name: "argocd-cmd-params-cm", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "argocd-cmd-params-cm"},
							Optional:             ptr(true),
							Items:                []corev1.KeyToPath{{Key: "applicationsetcontroller.profile.enabled", Path: "profiler.enabled"}},
						}}},
					},
				},
			},
		},
	}
}

func buildDeploymentRedis(ns string) appsv1.Deployment {
	labels := argoLabels("redis", "argocd-redis")
	podLabels := map[string]string{"app.kubernetes.io/name": "argocd-redis"}
	return appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-redis", Namespace: ns, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: podLabels},
				Spec: corev1.PodSpec{
					ServiceAccountName: "argocd-redis",
					NodeSelector:       map[string]string{"kubernetes.io/os": "linux"},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptr(true),
						RunAsUser:    ptr(int64(999)),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:            "secret-init",
							Image:           argocdImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"argocd", "admin", "redis-initial-password"},
							SecurityContext: restrictedSecCtx(),
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "redis",
							Image:           redisImage,
							ImagePullPolicy: corev1.PullAlways,
							Args:            []string{"--save", "", "--appendonly", "no", "--requirepass $(REDIS_PASSWORD)"},
							Env: []corev1.EnvVar{
								secretRef("REDIS_PASSWORD", "argocd-redis", "auth"),
							},
							Ports: []corev1.ContainerPort{{ContainerPort: 6379}},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: ptr(false),
								Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
								ReadOnlyRootFilesystem:   ptr(true),
							},
						},
					},
				},
			},
		},
	}
}

func buildDeploymentRepoServer(ns string) appsv1.Deployment {
	labels := argoLabels("repo-server", "argocd-repo-server")
	podLabels := map[string]string{"app.kubernetes.io/name": "argocd-repo-server"}
	return appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-repo-server", Namespace: ns, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: podLabels},
				Spec: corev1.PodSpec{
					ServiceAccountName:           "argocd-repo-server",
					AutomountServiceAccountToken: ptr(false),
					NodeSelector:                 map[string]string{"kubernetes.io/os": "linux"},
					InitContainers: []corev1.Container{
						{
							Name:    "copyutil",
							Image:   argocdImage,
							Command: []string{"sh", "-c"},
							Args:    []string{"/bin/cp /usr/local/bin/argocd /var/run/argocd/argocd && /bin/ln -sf /var/run/argocd/argocd /var/run/argocd/argocd-cmp-server"},
							SecurityContext: restrictedSecCtx(),
							VolumeMounts: []corev1.VolumeMount{
								{Name: "var-files", MountPath: "/var/run/argocd"},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "argocd-repo-server",
							Image:           argocdImage,
							ImagePullPolicy: corev1.PullAlways,
							Args:            []string{"/usr/local/bin/argocd-repo-server"},
							Env: []corev1.EnvVar{
								secretRef("REDIS_PASSWORD", "argocd-redis", "auth"),
								cmRef("ARGOCD_RECONCILIATION_TIMEOUT", "argocd-cm", "timeout.reconciliation"),
								cmRef("ARGOCD_REPO_SERVER_LOGFORMAT", "argocd-cmd-params-cm", "reposerver.log.format"),
								cmRef("ARGOCD_REPO_SERVER_LOGLEVEL", "argocd-cmd-params-cm", "reposerver.log.level"),
								cmRef("ARGOCD_REPO_SERVER_PARALLELISM_LIMIT", "argocd-cmd-params-cm", "reposerver.parallelism.limit"),
								cmRef("ARGOCD_REPO_SERVER_DISABLE_TLS", "argocd-cmd-params-cm", "reposerver.disable.tls"),
								cmRef("ARGOCD_REPO_CACHE_EXPIRATION", "argocd-cmd-params-cm", "reposerver.repo.cache.expiration"),
								cmRef("REDIS_SERVER", "argocd-cmd-params-cm", "redis.server"),
								cmRef("REDIS_COMPRESSION", "argocd-cmd-params-cm", "redis.compression"),
								cmRef("REDISDB", "argocd-cmd-params-cm", "redis.db"),
								cmRef("ARGOCD_DEFAULT_CACHE_EXPIRATION", "argocd-cmd-params-cm", "reposerver.default.cache.expiration"),
								cmRef("ARGOCD_REPO_SERVER_OTLP_ADDRESS", "argocd-cmd-params-cm", "otlp.address"),
								cmRef("ARGOCD_GIT_MODULES_ENABLED", "argocd-cmd-params-cm", "reposerver.enable.git.submodule"),
								cmRef("ARGOCD_GRPC_MAX_SIZE_MB", "argocd-cmd-params-cm", "reposerver.grpc.max.size"),
								cmRef("GRPC_ENABLE_TXT_SERVICE_CONFIG", "argocd-cmd-params-cm", "reposerver.grpc.enable.txt.service.config"),
								{Name: "HELM_CACHE_HOME", Value: "/helm-working-dir"},
								{Name: "HELM_CONFIG_HOME", Value: "/helm-working-dir"},
								{Name: "HELM_DATA_HOME", Value: "/helm-working-dir"},
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 8081},
								{ContainerPort: 8084},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler:        corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/healthz?full=true", Port: intstr.FromInt32(8084)}},
								InitialDelaySeconds: 30,
								PeriodSeconds:       30,
								FailureThreshold:    3,
								TimeoutSeconds:      5,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler:        corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/healthz", Port: intstr.FromInt32(8084)}},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							SecurityContext: restrictedSecCtx(),
							VolumeMounts: []corev1.VolumeMount{
								{Name: "ssh-known-hosts", MountPath: "/app/config/ssh"},
								{Name: "tls-certs", MountPath: "/app/config/tls"},
								{Name: "gpg-keys", MountPath: "/app/config/gpg/source"},
								{Name: "gpg-keyring", MountPath: "/app/config/gpg/keys"},
								{Name: "argocd-repo-server-tls", MountPath: "/app/config/reposerver/tls"},
								{Name: "tmp", MountPath: "/tmp"},
								{Name: "helm-working-dir", MountPath: "/helm-working-dir"},
								{Name: "plugins", MountPath: "/home/argocd/cmp-server/plugins"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{Name: "ssh-known-hosts", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "argocd-ssh-known-hosts-cm"}}}},
						{Name: "tls-certs", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "argocd-tls-certs-cm"}}}},
						{Name: "gpg-keys", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "argocd-gpg-keys-cm"}}}},
						{Name: "gpg-keyring", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "tmp", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "helm-working-dir", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "plugins", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "var-files", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "argocd-repo-server-tls", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
							SecretName: "argocd-repo-server-tls",
							Optional:   ptr(true),
							Items: []corev1.KeyToPath{
								{Key: "tls.crt", Path: "tls.crt"},
								{Key: "tls.key", Path: "tls.key"},
								{Key: "ca.crt", Path: "ca.crt"},
							},
						}}},
					},
				},
			},
		},
	}
}

func buildStatefulSetApplicationController(ns string) appsv1.StatefulSet {
	labels := argoLabels("application-controller", "argocd-application-controller")
	podLabels := map[string]string{"app.kubernetes.io/name": "argocd-application-controller"}
	return appsv1.StatefulSet{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "StatefulSet"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-application-controller", Namespace: ns, Labels: labels},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    ptr(int32(1)),
			ServiceName: "argocd-application-controller",
			Selector:    &metav1.LabelSelector{MatchLabels: podLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: podLabels},
				Spec: corev1.PodSpec{
					ServiceAccountName: "argocd-application-controller",
					NodeSelector:       map[string]string{"kubernetes.io/os": "linux"},
					Containers: []corev1.Container{
						{
							Name:            "argocd-application-controller",
							Image:           argocdImage,
							ImagePullPolicy: corev1.PullAlways,
							Args:            []string{"/usr/local/bin/argocd-application-controller"},
							WorkingDir:      "/home/argocd",
							Env: []corev1.EnvVar{
								secretRef("REDIS_PASSWORD", "argocd-redis", "auth"),
								{Name: "ARGOCD_CONTROLLER_REPLICAS", Value: "1"},
								{Name: "KUBECACHEDIR", Value: "/tmp/kubecache"},
								cmRef("ARGOCD_RECONCILIATION_TIMEOUT", "argocd-cm", "timeout.reconciliation"),
								cmRef("ARGOCD_HARD_RECONCILIATION_TIMEOUT", "argocd-cm", "timeout.hard.reconciliation"),
								cmRef("ARGOCD_RECONCILIATION_JITTER", "argocd-cm", "timeout.reconciliation.jitter"),
								cmRef("ARGOCD_APPLICATION_CONTROLLER_REPO_SERVER", "argocd-cmd-params-cm", "repo.server"),
								cmRef("ARGOCD_APPLICATION_CONTROLLER_STATUS_PROCESSORS", "argocd-cmd-params-cm", "controller.status.processors"),
								cmRef("ARGOCD_APPLICATION_CONTROLLER_OPERATION_PROCESSORS", "argocd-cmd-params-cm", "controller.operation.processors"),
								cmRef("ARGOCD_APPLICATION_CONTROLLER_LOGFORMAT", "argocd-cmd-params-cm", "controller.log.format"),
								cmRef("ARGOCD_APPLICATION_CONTROLLER_LOGLEVEL", "argocd-cmd-params-cm", "controller.log.level"),
								cmRef("ARGOCD_APPLICATION_CONTROLLER_SELF_HEAL_TIMEOUT_SECONDS", "argocd-cmd-params-cm", "controller.self.heal.timeout.seconds"),
								cmRef("ARGOCD_APPLICATION_CONTROLLER_REPO_SERVER_TIMEOUT_SECONDS", "argocd-cmd-params-cm", "controller.repo.server.timeout.seconds"),
								cmRef("ARGOCD_APPLICATION_CONTROLLER_REPO_SERVER_PLAINTEXT", "argocd-cmd-params-cm", "controller.repo.server.plaintext"),
								cmRef("ARGOCD_APPLICATION_CONTROLLER_KUBECTL_PARALLELISM_LIMIT", "argocd-cmd-params-cm", "controller.kubectl.parallelism.limit"),
								cmRef("ARGOCD_APPLICATION_CONTROLLER_SERVER_SIDE_DIFF", "argocd-cmd-params-cm", "controller.diff.server.side"),
								cmRef("ARGOCD_APP_STATE_CACHE_EXPIRATION", "argocd-cmd-params-cm", "controller.app.state.cache.expiration"),
								cmRef("REDIS_SERVER", "argocd-cmd-params-cm", "redis.server"),
								cmRef("REDIS_COMPRESSION", "argocd-cmd-params-cm", "redis.compression"),
								cmRef("REDISDB", "argocd-cmd-params-cm", "redis.db"),
								cmRef("ARGOCD_DEFAULT_CACHE_EXPIRATION", "argocd-cmd-params-cm", "controller.default.cache.expiration"),
								cmRef("ARGOCD_APPLICATION_NAMESPACES", "argocd-cmd-params-cm", "application.namespaces"),
								cmRef("ARGOCD_CONTROLLER_SHARDING_ALGORITHM", "argocd-cmd-params-cm", "controller.sharding.algorithm"),
								cmRef("GRPC_ENABLE_TXT_SERVICE_CONFIG", "argocd-cmd-params-cm", "controller.grpc.enable.txt.service.config"),
							},
							Ports: []corev1.ContainerPort{{ContainerPort: 8082}},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler:        corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/healthz", Port: intstr.FromInt32(8082)}},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							SecurityContext: restrictedSecCtx(),
							VolumeMounts: []corev1.VolumeMount{
								{Name: "argocd-home", MountPath: "/home/argocd"},
								{Name: "argocd-cmd-params-cm", MountPath: "/home/argocd/params"},
								{Name: "argocd-repo-server-tls", MountPath: "/app/config/controller/tls"},
								{Name: "argocd-application-controller-tmp", MountPath: "/tmp"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{Name: "argocd-home", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "argocd-application-controller-tmp", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "argocd-repo-server-tls", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
							SecretName: "argocd-repo-server-tls",
							Optional:   ptr(true),
							Items: []corev1.KeyToPath{
								{Key: "tls.crt", Path: "tls.crt"},
								{Key: "tls.key", Path: "tls.key"},
								{Key: "ca.crt", Path: "ca.crt"},
							},
						}}},
						{Name: "argocd-cmd-params-cm", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "argocd-cmd-params-cm"},
							Optional:             ptr(true),
							Items:                []corev1.KeyToPath{{Key: "controller.profile.enabled", Path: "profiler.enabled"}},
						}}},
					},
				},
			},
		},
	}
}
