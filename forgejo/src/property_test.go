package main

import (
	"maps"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"pgregory.net/rapid"
)

func TestPropertyRenderSelectorsMatch(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := defaults
		v.RunnerCount = 0
		v.Domain = rapid.StringMatching(`[a-z][a-z0-9.-]{0,30}[a-z0-9]`).Draw(t, "domain")
		v.AdminUsername = rapid.StringMatching(`[a-z][a-z0-9_-]{0,20}`).Draw(t, "adminUsername")

		resources, err := render("forgejo", "forgejo", v)
		if err != nil {
			t.Fatalf("render failed: %v", err)
		}

		deps := findAll[appsv1.Deployment](resources)
		if len(deps) != 1 {
			t.Fatalf("want 1 Deployment, got %d", len(deps))
		}
		dep := deps[0]
		if dep.Spec.Selector == nil {
			t.Fatal("deployment selector is nil")
		}
		podLabels := dep.Spec.Template.Labels
		if !maps.Equal(dep.Spec.Selector.MatchLabels, podLabels) {
			t.Fatalf("selector %v != pod template labels %v",
				dep.Spec.Selector.MatchLabels, podLabels)
		}

		for _, svc := range findAll[corev1.Service](resources) {
			if !maps.Equal(svc.Spec.Selector, podLabels) {
				t.Fatalf("service %q selector %v != pod labels %v",
					svc.Name, svc.Spec.Selector, podLabels)
			}
		}
	})
}

func TestPropertyAppIniReflectsDomain(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		domain := rapid.StringMatching(`[a-z][a-z0-9.-]{0,40}[a-z0-9]`).Draw(t, "domain")
		ini, err := generateAppIni(domain)
		if err != nil {
			t.Fatalf("generateAppIni failed: %v", err)
		}
		for _, want := range []string{
			"DOMAIN = " + domain,
			"SSH_DOMAIN = " + domain,
			"ROOT_URL = https://" + domain + "/",
		} {
			if !strings.Contains(ini, want) {
				t.Errorf("app.ini missing %q", want)
			}
		}
		if strings.Contains(ini, "{{") {
			t.Error("app.ini contains unrendered template markers")
		}
	})
}

