package controllers

import (
	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	userConfigurationConfigMapName = "user-config"
	userConfigurationSecretName    = "user-secret"
)

type seedJobConfig struct {
	v1alpha2.SeedJob
	JobNames   []string `json:"jobNames,omitempty"`
	Username   string   `json:"username,omitempty"`
	Password   string   `json:"password,omitempty"`
	PrivateKey string   `json:"privateKey,omitempty"`
}

var (
	jenkinsCRName     = "jenkins-example"
	namespace         = "default"
	priorityClassName = ""

	mySeedJob = seedJobConfig{
		SeedJob: v1alpha2.SeedJob{
			ID:                    "jenkins-operator",
			CredentialID:          "jenkins-operator",
			JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
			Targets:               "cicd/jobs/*.jenkins",
			Description:           "Jenkins Operator repository",
			RepositoryBranch:      "master",
			RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
			PollSCM:               "1 1 1 1 1",
			UnstableOnDeprecation: true,
			BuildPeriodically:     "1 1 1 1 1",
			FailOnMissingPlugin:   true,
			IgnoreMissingFiles:    true,
			//AdditionalClasspath: can fail with the seed job agent
			GitHubPushTrigger: true,
		},
	}
	groovyScripts = v1alpha2.GroovyScripts{
		Customization: v1alpha2.Customization{
			Configurations: []v1alpha2.ConfigMapRef{
				{
					Name: userConfigurationConfigMapName,
				},
			},
			Secret: v1alpha2.SecretRef{
				Name: userConfigurationSecretName,
			},
		},
	}

	casc = v1alpha2.ConfigurationAsCode{
		Customization: v1alpha2.Customization{
			Configurations: []v1alpha2.ConfigMapRef{
				{
					Name: userConfigurationConfigMapName,
				},
			},
			Secret: v1alpha2.SecretRef{
				Name: userConfigurationSecretName,
			},
		},
	}
)

func createJenkinsCR(name, namespace string, seedJob *[]v1alpha2.SeedJob, groovyScripts v1alpha2.GroovyScripts, casc v1alpha2.ConfigurationAsCode, priorityClassName string) *v1alpha2.Jenkins {
	var seedJobs []v1alpha2.SeedJob
	if seedJob != nil {
		seedJobs = append(seedJobs, *seedJob...)
	}

	jenkins := &v1alpha2.Jenkins{
		TypeMeta: v1alpha2.JenkinsTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha2.JenkinsSpec{
			GroovyScripts:       groovyScripts,
			ConfigurationAsCode: casc,
			Master: v1alpha2.JenkinsMaster{
				Annotations: map[string]string{"test": "label"},
				Containers: []v1alpha2.Container{
					{
						Name: resources.JenkinsMasterContainerName,
						Env: []corev1.EnvVar{
							{
								Name:  "TEST_ENV",
								Value: "test_env_value",
							},
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/login",
									Port:   intstr.FromString("http"),
									Scheme: corev1.URISchemeHTTP,
								},
							},
							InitialDelaySeconds: int32(80),
							TimeoutSeconds:      int32(4),
							FailureThreshold:    int32(10),
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/login",
									Port:   intstr.FromString("http"),
									Scheme: corev1.URISchemeHTTP,
								},
							},
							InitialDelaySeconds: int32(80),
							TimeoutSeconds:      int32(4),
							FailureThreshold:    int32(10),
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "plugins-cache",
								MountPath: "/usr/share/jenkins/ref/plugins",
							},
						},
					},
					{
						Name:  "envoyproxy",
						Image: "envoyproxy/envoy-alpine:v1.14.1",
					},
				},
				Plugins: []v1alpha2.Plugin{
					{Name: "audit-trail", Version: "3.7"},
					{Name: "simple-theme-plugin", Version: "0.6"},
					{Name: "github", Version: "1.32.0"},
					{Name: "devoptics", Version: "1.1905", DownloadURL: "https://jenkins-updates.cloudbees.com/download/plugins/devoptics/1.1905/devoptics.hpi"},
				},
				PriorityClassName: priorityClassName,
				NodeSelector:      map[string]string{"kubernetes.io/os": "linux"},
				Volumes: []corev1.Volume{
					{
						Name: "plugins-cache",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
			SeedJobs: seedJobs,
			Service: v1alpha2.Service{
				Type: corev1.ServiceTypeNodePort,
				Port: constants.DefaultHTTPPortInt32,
			},
		},
	}
	jenkins.Spec.Roles = []rbacv1.RoleRef{
		{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     resources.GetResourceName(jenkins),
		},
	}
	return jenkins
}
