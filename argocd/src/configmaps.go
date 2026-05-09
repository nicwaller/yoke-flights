package main

import (
	_ "embed"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"gopkg.in/yaml.v3"
)

//go:embed ssh_known_hosts
var sshKnownHosts string

// filteredResource mirrors the upstream FilteredResource schema without importing argocd packages.
// See: github.com/argoproj/argo-cd/v3/util/settings.FilteredResource
type filteredResource struct {
	APIGroups  []string `yaml:"apiGroups,omitempty"`
	Kinds      []string `yaml:"kinds,omitempty"`
	Clusters   []string `yaml:"clusters,omitempty"`
	Namespaces []string `yaml:"namespaces,omitempty"`
}

// ignoreResourceUpdates mirrors the upstream ResourceIgnoreDifferences / IgnoreResourceUpdates schema.
// See: github.com/argoproj/argo-cd/v3/util/settings
type ignoreResourceUpdates struct {
	JSONPointers      []string `yaml:"jsonPointers,omitempty"`
	JQPathExpressions []string `yaml:"jqPathExpressions,omitempty"`
}

func mustMarshalYAML(v any) string {
	b, err := yaml.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func buildConfigMapArgocdCm(ns string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-cm", Namespace: ns},
		Data: map[string]string{
			"resource.exclusions": mustMarshalYAML([]filteredResource{
				// Network resources created by the Kubernetes control plane
				{APIGroups: []string{"", "discovery.k8s.io"}, Kinds: []string{"Endpoints", "EndpointSlice"}},
				// Internal Kubernetes resources
				{APIGroups: []string{"coordination.k8s.io"}, Kinds: []string{"Lease"}},
				// Internal Kubernetes Authz/Authn resources
				{APIGroups: []string{"authentication.k8s.io", "authorization.k8s.io"}, Kinds: []string{
					"SelfSubjectReview", "TokenReview", "LocalSubjectAccessReview",
					"SelfSubjectAccessReview", "SelfSubjectRulesReview", "SubjectAccessReview",
				}},
				// Intermediate Certificate Requests
				{APIGroups: []string{"certificates.k8s.io"}, Kinds: []string{"CertificateSigningRequest"}},
				{APIGroups: []string{"cert-manager.io"}, Kinds: []string{"CertificateRequest"}},
				// Cilium internal resources
				{APIGroups: []string{"cilium.io"}, Kinds: []string{"CiliumIdentity", "CiliumEndpoint", "CiliumEndpointSlice"}},
				// Kyverno intermediate and reporting resources
				{APIGroups: []string{"kyverno.io", "reports.kyverno.io", "wgpolicyk8s.io"}, Kinds: []string{
					"PolicyReport", "ClusterPolicyReport", "EphemeralReport", "ClusterEphemeralReport",
					"AdmissionReport", "ClusterAdmissionReport", "BackgroundScanReport",
					"ClusterBackgroundScanReport", "UpdateRequest",
				}},
			}),
			"resource.customizations.ignoreResourceUpdates.all": mustMarshalYAML(ignoreResourceUpdates{
				JSONPointers: []string{"/status"},
			}),
			"resource.customizations.ignoreResourceUpdates.ConfigMap": mustMarshalYAML(ignoreResourceUpdates{
				JQPathExpressions: []string{
					`.metadata.annotations."cluster-autoscaler.kubernetes.io/last-updated"`,
					`.metadata.annotations."control-plane.alpha.kubernetes.io/leader"`,
				},
			}),
			"resource.customizations.ignoreResourceUpdates.Endpoints": mustMarshalYAML(ignoreResourceUpdates{
				JSONPointers: []string{"/metadata", "/subsets"},
			}),
			"resource.customizations.ignoreResourceUpdates.apps_ReplicaSet": mustMarshalYAML(ignoreResourceUpdates{
				JQPathExpressions: []string{
					`.metadata.annotations."deployment.kubernetes.io/desired-replicas"`,
					`.metadata.annotations."deployment.kubernetes.io/max-replicas"`,
					`.metadata.annotations."rollout.argoproj.io/desired-replicas"`,
				},
			}),
			"resource.customizations.ignoreResourceUpdates.argoproj.io_Application": mustMarshalYAML(ignoreResourceUpdates{
				JQPathExpressions: []string{
					`.metadata.annotations."notified.notifications.argoproj.io"`,
					`.metadata.annotations."argocd.argoproj.io/refresh"`,
					`.metadata.annotations."argocd.argoproj.io/hydrate"`,
					`.operation`,
				},
			}),
			"resource.customizations.ignoreResourceUpdates.argoproj.io_Rollout": mustMarshalYAML(ignoreResourceUpdates{
				JQPathExpressions: []string{
					`.metadata.annotations."notified.notifications.argoproj.io"`,
				},
			}),
			"resource.customizations.ignoreResourceUpdates.autoscaling_HorizontalPodAutoscaler": mustMarshalYAML(ignoreResourceUpdates{
				JQPathExpressions: []string{
					`.metadata.annotations."autoscaling.alpha.kubernetes.io/behavior"`,
					`.metadata.annotations."autoscaling.alpha.kubernetes.io/conditions"`,
					`.metadata.annotations."autoscaling.alpha.kubernetes.io/metrics"`,
					`.metadata.annotations."autoscaling.alpha.kubernetes.io/current-metrics"`,
				},
			}),
			"resource.customizations.ignoreResourceUpdates.discovery.k8s.io_EndpointSlice": mustMarshalYAML(ignoreResourceUpdates{
				JSONPointers: []string{"/metadata", "/endpoints", "/ports"},
			}),
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
			"ssh_known_hosts": sshKnownHosts,
		},
	}
}

func buildConfigMapTlsCerts(ns string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-tls-certs-cm", Namespace: ns},
	}
}

func buildSecretArgocd(ns string) corev1.Secret {
	return corev1.Secret{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-secret", Namespace: ns},
		Type:       corev1.SecretTypeOpaque,
	}
}
