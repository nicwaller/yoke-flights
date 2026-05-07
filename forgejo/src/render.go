package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	forgejoImage    = "codeberg.org/forgejo/forgejo:15.0.1-rootless"
	forgejoUID      = 1000
	httpPort        = 3000
	sshListenPort   = 2222
	sshExternalPort = 22
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
		buildService(name+"-http", ns, labels, "http", httpPort, values.HTTPServiceType),
		buildService(name+"-ssh", ns, labels, "ssh", sshExternalPort, values.SSHServiceType),
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
			Replicas: ptr(int32(1)),
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					SecurityContext:               &corev1.PodSecurityContext{FSGroup: ptr(int64(forgejoUID))},
					TerminationGracePeriodSeconds: ptr(int64(60)),
					InitContainers: []corev1.Container{
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
					},
					Containers: []corev1.Container{
						{
							Name:  "forgejo",
							Image: forgejoImage,
							Env: append(envBase(),
								corev1.EnvVar{Name: "SSH_LISTEN_PORT", Value: strconv.Itoa(sshListenPort)},
								corev1.EnvVar{Name: "SSH_PORT", Value: strconv.Itoa(sshExternalPort)},
							),
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: httpPort},
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

func buildService(name, ns string, selector map[string]string, portName string, port int32, svcType string) corev1.Service {
	return corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceType(svcType),
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
