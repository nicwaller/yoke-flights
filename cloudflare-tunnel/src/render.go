package main

import (
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func render(name, ns string, values Values) ([]json.RawMessage, error) {
	if err := values.validate(); err != nil {
		return nil, err
	}

	labels := map[string]string{"app": name}

	objects := []any{
		buildTokenSecret(name, ns, values.Token),
		buildDeployment(name, ns, values.Image, labels),
	}

	result := make([]json.RawMessage, len(objects))
	for i, obj := range objects {
		b, err := json.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource: %w", err)
		}
		result[i] = b
	}
	return result, nil
}

func buildTokenSecret(name, ns, token string) corev1.Secret {
	return corev1.Secret{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Name: name + "-token", Namespace: ns},
		StringData: map[string]string{"token": token},
	}
}

func buildDeployment(name, ns, image string, labels map[string]string) appsv1.Deployment {
	return appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr(int32(1)),
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "cloudflared",
							Image: image,
							Args:  []string{"tunnel", "--no-autoupdate", "run"},
							Env: []corev1.EnvVar{
								{
									Name: "TUNNEL_TOKEN",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: name + "-token"},
											Key:                  "token",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func ptr[T any](v T) *T { return &v }
