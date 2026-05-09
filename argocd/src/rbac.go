package main

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func buildSA(name, ns string) corev1.ServiceAccount {
	return corev1.ServiceAccount{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
	}
}

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

func buildRoleServer(ns string) rbacv1.Role {
	return rbacv1.Role{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "Role"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-server", Namespace: ns},
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{""}, Resources: []string{"secrets", "configmaps"}, Verbs: []string{"create", "get", "list", "watch", "update", "patch", "delete"}},
			{APIGroups: []string{"argoproj.io"}, Resources: []string{"applications", "applicationsets", "appprojects"}, Verbs: []string{"create", "get", "list", "watch", "update", "delete", "patch"}},
			{APIGroups: []string{""}, Resources: []string{"events"}, Verbs: []string{"create", "list"}},
		},
	}
}

func buildClusterRoleServer() rbacv1.ClusterRole {
	return rbacv1.ClusterRole{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "ClusterRole"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-server"},
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"delete", "get", "patch"}},
			{APIGroups: []string{""}, Resources: []string{"events"}, Verbs: []string{"list", "create"}},
			{APIGroups: []string{""}, Resources: []string{"pods", "pods/log"}, Verbs: []string{"get"}},
			{APIGroups: []string{"argoproj.io"}, Resources: []string{"applications", "applicationsets"}, Verbs: []string{"get", "list", "update", "watch"}},
			{APIGroups: []string{"batch"}, Resources: []string{"jobs"}, Verbs: []string{"create"}},
			{APIGroups: []string{"argoproj.io"}, Resources: []string{"workflows"}, Verbs: []string{"create"}},
		},
	}
}

func buildClusterRoleBindingServer(ns string) rbacv1.ClusterRoleBinding {
	return rbacv1.ClusterRoleBinding{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "ClusterRoleBinding"},
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-server"},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "argocd-server"},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: "argocd-server", Namespace: ns}},
	}
}

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
