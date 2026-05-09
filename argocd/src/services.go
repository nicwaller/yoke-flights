package main

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

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
