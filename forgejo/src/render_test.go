package main

import (
	"encoding/json"
	"fmt"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestResourceTypes(t *testing.T) {
	resources, err := render("forgejo", "forgejo", defaults)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(findAll[appsv1.Deployment](resources)); n != 1 {
		t.Errorf("want 1 Deployment, got %d", n)
	}
	if n := len(findAll[corev1.PersistentVolumeClaim](resources)); n != 1 {
		t.Errorf("want 1 PVC, got %d", n)
	}
	if n := len(findAll[corev1.Service](resources)); n != 2 {
		t.Errorf("want 2 Services, got %d", n)
	}
	if n := len(findAll[corev1.Secret](resources)); n != 2 {
		t.Errorf("want 2 Secrets, got %d", n)
	}
}

func TestImage(t *testing.T) {
	resources, err := render("forgejo", "forgejo", defaults)
	if err != nil {
		t.Fatal(err)
	}
	dep := findOne[appsv1.Deployment](t, resources)
	if got := dep.Spec.Template.Spec.Containers[0].Image; got != forgejoImage {
		t.Fatalf("image: got %q, want %q", got, forgejoImage)
	}
}

func TestInvalidStorageSize(t *testing.T) {
	values := defaults
	values.StorageSize = "not-a-quantity"
	if _, err := render("forgejo", "forgejo", values); err == nil {
		t.Fatal("expected error for invalid storageSize")
	}
}

func TestAdminPasswordProvided(t *testing.T) {
	values := defaults
	values.AdminPassword = "mysecret"
	resources, err := render("forgejo", "forgejo", values)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range findAll[corev1.Secret](resources) {
		if s.Name == "forgejo-admin" {
			if got := s.StringData["password"]; got != "mysecret" {
				t.Fatalf("admin password: got %q, want %q", got, "mysecret")
			}
			return
		}
	}
	t.Fatal("admin secret not found")
}

func TestDeploymentLabels(t *testing.T) {
	resources, err := render("forgejo", "forgejo", defaults)
	if err != nil {
		t.Fatal(err)
	}
	dep := findOne[appsv1.Deployment](t, resources)
	if dep.Spec.Selector == nil {
		t.Fatal("deployment selector is nil")
	}
	if got := dep.Spec.Selector.MatchLabels["app"]; got != "forgejo" {
		t.Fatalf("selector label: got %q, want %q", got, "forgejo")
	}
	if got := dep.Spec.Template.Labels["app"]; got != "forgejo" {
		t.Fatalf("pod template label: got %q, want %q", got, "forgejo")
	}
}

func kindFor[T any]() string {
	switch any(*new(T)).(type) {
	case appsv1.Deployment:
		return "Deployment"
	case corev1.Service:
		return "Service"
	case corev1.Secret:
		return "Secret"
	case corev1.PersistentVolumeClaim:
		return "PersistentVolumeClaim"
	default:
		panic(fmt.Sprintf("unknown resource type %T", *new(T)))
	}
}

func findAll[T any](resources []json.RawMessage) []T {
	kind := kindFor[T]()
	var tm struct {
		Kind string `json:"kind"`
	}
	var result []T
	for _, r := range resources {
		if json.Unmarshal(r, &tm) == nil && tm.Kind == kind {
			var v T
			if json.Unmarshal(r, &v) == nil {
				result = append(result, v)
			}
		}
	}
	return result
}

func findOne[T any](t *testing.T, resources []json.RawMessage) T {
	t.Helper()
	all := findAll[T](resources)
	if len(all) == 0 {
		t.Fatalf("no %s found", kindFor[T]())
	}
	return all[0]
}

func findService(t *testing.T, resources []json.RawMessage, name string) corev1.Service {
	t.Helper()
	for _, svc := range findAll[corev1.Service](resources) {
		if svc.Name == name {
			return svc
		}
	}
	t.Fatalf("service %q not found", name)
	return corev1.Service{}
}
