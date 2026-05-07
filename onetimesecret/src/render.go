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
	otsImage   = "docker.io/onetimesecret/onetimesecret:v0.18.5"
	redisImage = "redis:7-alpine"
	otsPort    = int32(3000)
	redisPort  = int32(6379)
)

func render(name, ns string, values Values) ([]json.RawMessage, error) {
	if err := values.validate(); err != nil {
		return nil, err
	}

	redisStorageQty, err := resource.ParseQuantity(values.RedisStorageSize)
	if err != nil {
		return nil, fmt.Errorf("invalid redisStorageSize: %w", err)
	}

	appLabels := map[string]string{"app": name}
	redisLabels := map[string]string{"app": name + "-redis"}
	redisURL := "redis://" + name + "-redis:6379/0"

	objects := []any{
		buildSmtpSecret(name, ns, values.SmtpPassword),
		buildRedisPVC(name, ns, values.RedisStorageClass, redisStorageQty),
		buildRedisDeployment(name, ns, redisLabels),
		buildRedisService(name, ns, redisLabels),
		buildOTSDeployment(name, ns, values, appLabels, redisURL),
		buildOTSService(name, ns, appLabels, values.ServiceType, int32(values.Port)),
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

func buildSmtpSecret(name, ns, smtpPassword string) corev1.Secret {
	return corev1.Secret{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Name: name + "-smtp", Namespace: ns},
		StringData: map[string]string{"password": smtpPassword},
	}
}

func buildRedisPVC(name, ns, storageClass string, qty resource.Quantity) corev1.PersistentVolumeClaim {
	return corev1.PersistentVolumeClaim{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolumeClaim"},
		ObjectMeta: metav1.ObjectMeta{Name: name + "-redis-data", Namespace: ns},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &storageClass,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: qty},
			},
		},
	}
}

func buildRedisDeployment(name, ns string, labels map[string]string) appsv1.Deployment {
	return appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: name + "-redis", Namespace: ns, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr(int32(1)),
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "redis",
							Image:   redisImage,
							Command: []string{"redis-server", "--save", "60", "1", "--loglevel", "warning"},
							Ports:   []corev1.ContainerPort{{Name: "redis", ContainerPort: redisPort}},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"redis-cli", "ping"},
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "data", MountPath: "/data"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: name + "-redis-data",
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildRedisService(name, ns string, selector map[string]string) corev1.Service {
	return corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: name + "-redis", Namespace: ns},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Name:       "redis",
					Port:       redisPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("redis"),
				},
			},
		},
	}
}

func buildOTSDeployment(name, ns string, values Values, labels map[string]string, redisURL string) appsv1.Deployment {
	ssl := boolToString(values.SSL)
	authSignup := boolToString(values.AuthSignup)
	authSignin := boolToString(values.AuthSignin)

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
							Name:  "onetimesecret",
							Image: otsImage,
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: otsPort},
							},
							Env: []corev1.EnvVar{
								{Name: "COLONEL", Value: values.Colonel},
								{Name: "SSL", Value: ssl},
								{Name: "SMTP_HOST", Value: values.SmtpHost},
								{Name: "SMTP_PORT", Value: strconv.Itoa(values.SmtpPort)},
								{Name: "SMTP_USERNAME", Value: values.SmtpUsername},
								{
									Name: "SMTP_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: name + "-smtp"},
										Key:                  "password",
									}},
								},
								{Name: "FROM_EMAIL", Value: values.FromEmail},
								{Name: "TO_EMAIL", Value: values.Colonel},
								{Name: "AUTH_SIGNUP", Value: authSignup},
								{Name: "AUTH_SIGNIN", Value: authSignin},
								{Name: "HOST", Value: values.Domain},
								{Name: "REDIS_URL", Value: redisURL},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
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
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
								FailureThreshold:    5,
							},
						},
					},
				},
			},
		},
	}
}

func buildOTSService(name, ns string, selector map[string]string, svcType string, externalPort int32) corev1.Service {
	return corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceType(svcType),
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       externalPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("http"),
				},
			},
		},
	}
}
