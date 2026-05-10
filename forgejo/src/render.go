package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	forgejoImage    = "codeberg.org/forgejo/forgejo:15.0.1-rootless"
	runnerImage     = "data.forgejo.org/forgejo/runner:12"
	kubectlImage    = "bitnami/kubectl"
	forgejoUID      = 1000
	httpListenPort  = 8080
	httpExternalPort = 80
	sshListenPort   = 2222
	sshExternalPort = 2222
)

const initDirsScript = `mkdir -p /data/git/.ssh /data/gitea/conf /tmp/gitea &&
chmod -R 700 /data/git/.ssh &&
chmod ug+rwx /tmp/gitea &&
cp /config/app.ini /data/gitea/conf/app.ini`

const configureGiteaScriptFmt = `gitea migrate &&
if ! gitea admin user create --admin \
  --username "$GITEA_ADMIN_USERNAME" \
  --password "$GITEA_ADMIN_PASSWORD" \
  --email "admin@%s" \
  --must-change-password=false; then
  gitea admin user change-password \
    --username "$GITEA_ADMIN_USERNAME" \
    --password "$GITEA_ADMIN_PASSWORD" \
    --must-change-password=false
fi`

// runnerEntrypointScript registers this pod as an ephemeral runner via the Forgejo HTTP API
// then immediately runs one-job using the returned credentials.
// Requires FORGEJO_URL, ADMIN_USERNAME, ADMIN_PASSWORD, and POD_NAME env vars.
const runnerEntrypointScript = `set -e
mkdir -p /tmp/runner-work
cat > /tmp/runner-config.yaml << 'YAML'
runner:
  labels:
    - "self-hosted:host"
host:
  workdir_parent: "/tmp/runner-work"
YAML
wget -qO /tmp/runner-response \
  --header "Content-Type: application/json" \
  --header "Authorization: Basic $(printf '%s:%s' "$ADMIN_USERNAME" "$ADMIN_PASSWORD" | base64 | tr -d '\n')" \
  --post-data "{\"name\":\"$POD_NAME\",\"ephemeral\":true}" \
  "$FORGEJO_URL/api/v1/admin/actions/runners"
UUID=$(sed 's/.*"uuid":"\([^"]*\)".*/\1/' /tmp/runner-response)
sed 's/.*"token":"\([^"]*\)".*/\1/' /tmp/runner-response > /tmp/runner-token
exec forgejo-runner one-job --wait \
  --config /tmp/runner-config.yaml \
  --url "$FORGEJO_URL/" \
  --uuid "$UUID" \
  --token-url "file:///tmp/runner-token"`

func render(name, ns string, values Values) ([]json.RawMessage, error) {
	if err := values.validate(); err != nil {
		return nil, err
	}

	appIni, err := resolveAppIni(name, ns, values.Domain)
	if err != nil {
		return nil, err
	}

	adminPassword, err := resolveAdminPassword(name, ns, values.AdminPassword)
	if err != nil {
		return nil, err
	}

	storageQty, err := resource.ParseQuantity(values.StorageSize)
	if err != nil {
		return nil, fmt.Errorf("invalid storageSize: %w", err)
	}

	labels := map[string]string{"app": name}

	objects := []any{
		buildConfigSecret(name, ns, appIni),
		buildAdminSecret(name, ns, values.AdminUsername, adminPassword),
		buildPVC(name, ns, values.StorageClass, storageQty),
		buildDeployment(name, ns, values.Domain, labels),
		buildService(name+"-http", ns, labels, "http", httpExternalPort),
		buildService(name+"-ssh", ns, labels, "ssh", sshExternalPort),
	}

	if values.RunnerCount > 0 {
		objects = append(objects,
			buildRunnerJob(name, ns, values.RunnerCount),
			buildRunnerPurgeSA(name, ns),
			buildRunnerPurgeRole(name, ns),
			buildRunnerPurgeRoleBinding(name, ns),
			buildRunnerPurgeCronJob(name, ns),
		)
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

func buildConfigSecret(name, ns, appIni string) corev1.Secret {
	return corev1.Secret{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Name: name + "-config", Namespace: ns},
		StringData: map[string]string{"app.ini": appIni},
	}
}

func buildAdminSecret(name, ns, username, password string) corev1.Secret {
	return corev1.Secret{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Name: name + "-admin", Namespace: ns},
		StringData: map[string]string{
			"username": username,
			"password": password,
		},
	}
}

func buildPVC(name, ns, storageClass string, qty resource.Quantity) corev1.PersistentVolumeClaim {
	return corev1.PersistentVolumeClaim{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolumeClaim"},
		ObjectMeta: metav1.ObjectMeta{Name: name + "-data", Namespace: ns},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &storageClass,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: qty},
			},
		},
	}
}

func buildDeployment(name, ns, domain string, labels map[string]string) appsv1.Deployment {
	return appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas:             ptr(int32(1)),
			RevisionHistoryLimit: ptr(int32(0)),
			Strategy:             appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			Selector:             &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					SecurityContext:               &corev1.PodSecurityContext{FSGroup: ptr(int64(forgejoUID))},
					TerminationGracePeriodSeconds: ptr(int64(60)),
					InitContainers: buildInitContainers(name, ns, domain),
					Containers: []corev1.Container{
						{
							Name:  "forgejo",
							Image: forgejoImage,
							Env: append(envBase(),
								corev1.EnvVar{Name: "SSH_LISTEN_PORT", Value: strconv.Itoa(sshListenPort)},
								corev1.EnvVar{Name: "SSH_PORT", Value: strconv.Itoa(sshExternalPort)},
							),
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: httpListenPort},
								{Name: "ssh", ContainerPort: sshListenPort},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/api/healthz",
										Port: intstr.FromString("http"),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromString("http"),
									},
								},
								InitialDelaySeconds: 200,
								PeriodSeconds:       10,
								FailureThreshold:    10,
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "data", MountPath: "/data"},
								{Name: "temp", MountPath: "/tmp"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: name + "-data"},
							},
						},
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{SecretName: name + "-config"},
							},
						},
						{
							Name:         "temp",
							VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
						},
					},
				},
			},
		},
	}
}

func buildInitContainers(name, ns, domain string) []corev1.Container {
	return []corev1.Container{
		{
			Name:    "init-directories",
			Image:   forgejoImage,
			Command: []string{"/bin/sh", "-c", initDirsScript},
			Env:     envBase(),
			VolumeMounts: []corev1.VolumeMount{
				{Name: "data", MountPath: "/data"},
				{Name: "config", MountPath: "/config"},
				{Name: "temp", MountPath: "/tmp"},
			},
		},
		{
			Name:    "configure-gitea",
			Image:   forgejoImage,
			Command: []string{"/bin/sh", "-c", fmt.Sprintf(configureGiteaScriptFmt, domain)},
			Env: append(envBase(),
				corev1.EnvVar{
					Name: "GITEA_ADMIN_USERNAME",
					ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: name + "-admin"},
						Key:                  "username",
					}},
				},
				corev1.EnvVar{
					Name: "GITEA_ADMIN_PASSWORD",
					ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: name + "-admin"},
						Key:                  "password",
					}},
				},
			),
			SecurityContext: &corev1.SecurityContext{RunAsUser: ptr(int64(forgejoUID))},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "data", MountPath: "/data"},
				{Name: "temp", MountPath: "/tmp"},
			},
		},
	}
}

func buildRunnerJob(name, ns string, count int) batchv1.Job {
	labels := map[string]string{"app": name + "-runner"}
	forgejoURL := fmt.Sprintf("http://%s-http.%s.svc.cluster.local:%d", name, ns, httpExternalPort)
	script := "FORGEJO_URL=" + forgejoURL + "\n" + runnerEntrypointScript
	return batchv1.Job{
		TypeMeta:   metav1.TypeMeta{APIVersion: "batch/v1", Kind: "Job"},
		ObjectMeta: metav1.ObjectMeta{Name: name + "-runner", Namespace: ns, Labels: labels},
		Spec: batchv1.JobSpec{
			Parallelism:             ptr(int32(count)),
			Completions:             ptr(int32(1<<31 - 1)),
			TTLSecondsAfterFinished: ptr(int32(60)),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:    "runner",
							Image:   runnerImage,
							Command: []string{"/bin/sh", "-c", script},
							Env: []corev1.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
									},
								},
								{
									Name: "ADMIN_USERNAME",
									ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: name + "-admin"},
										Key:                  "username",
									}},
								},
								{
									Name: "ADMIN_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: name + "-admin"},
										Key:                  "password",
									}},
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildRunnerPurgeSA(name, ns string) corev1.ServiceAccount {
	return corev1.ServiceAccount{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
		ObjectMeta: metav1.ObjectMeta{Name: name + "-runner-purge", Namespace: ns},
	}
}

func buildRunnerPurgeRole(name, ns string) rbacv1.Role {
	return rbacv1.Role{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "Role"},
		ObjectMeta: metav1.ObjectMeta{Name: name + "-runner-purge", Namespace: ns},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"list", "delete"},
			},
		},
	}
}

func buildRunnerPurgeRoleBinding(name, ns string) rbacv1.RoleBinding {
	return rbacv1.RoleBinding{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "RoleBinding"},
		ObjectMeta: metav1.ObjectMeta{Name: name + "-runner-purge", Namespace: ns},
		Subjects: []rbacv1.Subject{
			{Kind: "ServiceAccount", Name: name + "-runner-purge", Namespace: ns},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name + "-runner-purge",
		},
	}
}

func buildRunnerPurgeCronJob(name, ns string) batchv1.CronJob {
	script := fmt.Sprintf(`CUTOFF=$(date -d '5 minutes ago' +%%s)
kubectl get pods -n %s -l app=%s-runner --field-selector=status.phase=Succeeded \
  -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.status.containerStatuses[0].state.terminated.finishedAt}{"\n"}{end}' | \
while read -r pod finished; do
  [ -z "$finished" ] && continue
  finished_epoch=$(date -d "$finished" +%%s 2>/dev/null) || continue
  if [ "$finished_epoch" -lt "$CUTOFF" ]; then kubectl delete pod -n %s "$pod"; fi
done`, ns, name, ns)

	return batchv1.CronJob{
		TypeMeta:   metav1.TypeMeta{APIVersion: "batch/v1", Kind: "CronJob"},
		ObjectMeta: metav1.ObjectMeta{Name: name + "-runner-purge", Namespace: ns},
		Spec: batchv1.CronJobSpec{
			Schedule:                   "*/2 * * * *",
			ConcurrencyPolicy:          batchv1.ForbidConcurrent,
			SuccessfulJobsHistoryLimit: ptr(int32(1)),
			FailedJobsHistoryLimit:     ptr(int32(1)),
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					TTLSecondsAfterFinished: ptr(int32(60)),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							ServiceAccountName: name + "-runner-purge",
							RestartPolicy:      corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:    "purge",
									Image:   kubectlImage,
									Command: []string{"/bin/sh", "-c", script},
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildService(name, ns string, selector map[string]string, portName string, port int32) corev1.Service {
	return corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Name:       portName,
					Port:       port,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(portName),
				},
			},
		},
	}
}
