package main

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

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
