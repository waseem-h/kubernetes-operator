package v1alpha2

import (
	"errors"
	"testing"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMakeSemanticVersion(t *testing.T) {
	t.Run("only major version specified", func(t *testing.T) {
		got := makeSemanticVersion("1")
		assert.Equal(t, got, "v1.0.0")
	})

	t.Run("major and minor version specified", func(t *testing.T) {
		got := makeSemanticVersion("1.2")
		assert.Equal(t, got, "v1.2.0")
	})

	t.Run("major,minor and patch version specified", func(t *testing.T) {
		got := makeSemanticVersion("1.2.3")
		assert.Equal(t, got, "v1.2.3")
	})

	t.Run("semantic versions begin with a leading v and no patch version", func(t *testing.T) {
		got := makeSemanticVersion("v2.5")
		assert.Equal(t, got, "v2.5.0")
	})

	t.Run("semantic versions with prerelease versions", func(t *testing.T) {
		got := makeSemanticVersion("2.1.2-alpha.1")
		assert.Equal(t, got, "v2.1.2-alpha.1")
	})

	t.Run("semantic versions with prerelease versions", func(t *testing.T) {
		got := makeSemanticVersion("0.11.2-9.c8b45b8bb173")
		assert.Equal(t, got, "v0.11.2-9.c8b45b8bb173")
	})

	t.Run("semantic versions with build suffix", func(t *testing.T) {
		got := makeSemanticVersion("1.7.9+meta")
		assert.Equal(t, got, "v1.7.9")
	})

	t.Run("invalid semantic version", func(t *testing.T) {
		got := makeSemanticVersion("google-login-1.2")
		assert.Equal(t, got, "")
	})
}

func TestCompareVersions(t *testing.T) {
	t.Run("Plugin Version lies between first and last version", func(t *testing.T) {
		got := compareVersions("1.2", "1.6", "1.4")
		assert.Equal(t, got, true)
	})
	t.Run("Plugin Version is greater than the last version", func(t *testing.T) {
		got := compareVersions("1", "2", "3")
		assert.Equal(t, got, false)
	})
	t.Run("Plugin Version is less than the first version", func(t *testing.T) {
		got := compareVersions("1.4", "2.5", "1.1")
		assert.Equal(t, got, false)
	})

	t.Run("Plugins Versions have prerelease version and it lies between first and last version", func(t *testing.T) {
		got := compareVersions("1.2.1-alpha", "1.2.1", "1.2.1-beta")
		assert.Equal(t, got, true)
	})

	t.Run("Plugins Versions have prerelease version and it is greater than the last version", func(t *testing.T) {
		got := compareVersions("v2.2.1-alpha", "v2.5.1-beta.1", "v2.5.1-beta.2")
		assert.Equal(t, got, false)
	})
}

func TestValidate(t *testing.T) {
	t.Run("Validating when plugins data file is not fetched", func(t *testing.T) {
		userplugins := []Plugin{{Name: "script-security", Version: "1.77"}, {Name: "git-client", Version: "3.9"}, {Name: "git", Version: "4.8.1"}, {Name: "plain-credentials", Version: "1.7"}}
		jenkinscr := *createJenkinsCR(userplugins, true)
		got := jenkinscr.ValidateCreate()
		assert.Equal(t, got, errors.New("plugins data has not been fetched"))
	})

	isInitialized := make(chan bool)
	go PluginsMgr.FetchPluginData(isInitialized)
	if <-isInitialized {
		t.Run("Validating a Jenkins CR with plugins not having security warnings and validation is turned on", func(t *testing.T) {
			userplugins := []Plugin{{Name: "script-security", Version: "1.77"}, {Name: "git-client", Version: "3.9"}, {Name: "git", Version: "4.8.1"}, {Name: "plain-credentials", Version: "1.7"}}
			jenkinscr := *createJenkinsCR(userplugins, true)
			got := jenkinscr.ValidateCreate()
			assert.Nil(t, got)
		})

		t.Run("Validating a Jenkins CR with some of the plugins having security warnings and validation is turned on", func(t *testing.T) {
			userplugins := []Plugin{{Name: "google-login", Version: "1.2"}, {Name: "mailer", Version: "1.1"}, {Name: "git", Version: "4.8.1"}, {Name: "command-launcher", Version: "1.6"}, {Name: "workflow-cps", Version: "2.59"}}
			jenkinscr := *createJenkinsCR(userplugins, true)
			got := jenkinscr.ValidateCreate()
			assert.Equal(t, got, errors.New("security vulnerabilities detected in the following user-defined plugins: \nworkflow-cps:2.59\ngoogle-login:1.2\nmailer:1.1"))
		})

		t.Run("Updating a Jenkins CR with some of the plugins having security warnings and validation is turned on", func(t *testing.T) {
			userplugins := []Plugin{{Name: "google-login", Version: "1.2"}, {Name: "mailer", Version: "1.1"}, {Name: "git", Version: "4.8.1"}, {Name: "command-launcher", Version: "1.6"}, {Name: "workflow-cps", Version: "2.59"}}
			oldjenkinscr := *createJenkinsCR(userplugins, true)

			userplugins = []Plugin{{Name: "handy-uri-templates-2-api", Version: "2.1.8-1.0"}, {Name: "resource-disposer", Version: "0.8"}, {Name: "jjwt-api", Version: "0.11.2-9.c8b45b8bb173"}, {Name: "blueocean-github-pipeline", Version: "1.2.0-beta-3"}, {Name: "ghprb", Version: "1.39"}}
			newjenkinscr := *createJenkinsCR(userplugins, true)
			got := newjenkinscr.ValidateUpdate(&oldjenkinscr)
			assert.Equal(t, got, errors.New("security vulnerabilities detected in the following user-defined plugins: \nresource-disposer:0.8\nblueocean-github-pipeline:1.2.0-beta-3\nghprb:1.39"))
		})

		t.Run("Validation is turned off", func(t *testing.T) {
			userplugins := []Plugin{{Name: "google-login", Version: "1.2"}, {Name: "mailer", Version: "1.1"}, {Name: "git", Version: "4.8.1"}, {Name: "command-launcher", Version: "1.6"}, {Name: "workflow-cps", Version: "2.59"}}
			jenkinscr := *createJenkinsCR(userplugins, false)
			got := jenkinscr.ValidateCreate()
			assert.Nil(t, got)
		})
	} else {
		t.Fatal("Plugin Data File is not Downloaded")
	}
}

func TestFetchPluginData(t *testing.T) {
	t.Run("Timeout error while downloading plugins data file", func(t *testing.T) {
		pluginsDataMgr := *NewPluginsDataManager()
		pluginsDataMgr.Timeout = time.Duration(1) * time.Nanosecond
		got := pluginsDataMgr.download()
		assert.NotNil(t, got)
	})
	t.Run("Successfully fetching plugins data file", func(t *testing.T) {
		isInitialized := make(chan bool)
		pluginsDataMgr := *NewPluginsDataManager()
		go pluginsDataMgr.FetchPluginData(isInitialized)
		assert.Equal(t, <-isInitialized, true)
	})
}

func createJenkinsCR(userPlugins []Plugin, validateSecurityWarnings bool) *Jenkins {
	jenkins := &Jenkins{
		TypeMeta: JenkinsTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      "Jenkins",
			Namespace: "test",
		},
		Spec: JenkinsSpec{
			Master: JenkinsMaster{
				Annotations: map[string]string{"test": "label"},
				Plugins:     userPlugins,
			},
			ValidateSecurityWarnings: validateSecurityWarnings,
			Service: Service{
				Type: corev1.ServiceTypeNodePort,
				Port: constants.DefaultHTTPPortInt32,
			},
		},
	}

	return jenkins
}
