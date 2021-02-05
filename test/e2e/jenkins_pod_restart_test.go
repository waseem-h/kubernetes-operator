package e2e

import (
	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	// +kubebuilder:scaffold:imports
)

var _ = Describe("Jenkins controller", func() {

	const (
		jenkinsCRName     = e2e
		priorityClassName = ""
	)

	var (
		namespace     *corev1.Namespace
		jenkins       *v1alpha2.Jenkins
		groovyScripts = v1alpha2.GroovyScripts{
			Customization: v1alpha2.Customization{
				Configurations: []v1alpha2.ConfigMapRef{},
			},
		}
		casc = v1alpha2.ConfigurationAsCode{
			Customization: v1alpha2.Customization{
				Configurations: []v1alpha2.ConfigMapRef{},
			},
		}
	)

	BeforeEach(func() {
		namespace = createNamespace()

		configureAuthorizationToUnSecure(namespace.Name, userConfigurationConfigMapName)
		jenkins = createJenkinsCR(jenkinsCRName, namespace.Name, nil, groovyScripts, casc, priorityClassName)
	})

	AfterEach(func() {
		destroyNamespace(namespace)
	})

	Context("when restarting Jenkins master pod", func() {
		It("new Jenkins Master pod should be created", func() {
			waitForJenkinsBaseConfigurationToComplete(jenkins)
			restartJenkinsMasterPod(jenkins)
			waitForRecreateJenkinsMasterPod(jenkins)
			checkBaseConfigurationCompleteTimeIsNotSet(jenkins)
			waitForJenkinsBaseConfigurationToComplete(jenkins)
		})
	})
})

var _ = Describe("Jenkins controller", func() {

	const (
		jenkinsCRName     = e2e
		priorityClassName = ""
	)

	var (
		namespace     *corev1.Namespace
		jenkins       *v1alpha2.Jenkins
		groovyScripts = v1alpha2.GroovyScripts{
			Customization: v1alpha2.Customization{
				Configurations: []v1alpha2.ConfigMapRef{
					{
						Name: userConfigurationConfigMapName,
					},
				},
			},
		}
		casc = v1alpha2.ConfigurationAsCode{
			Customization: v1alpha2.Customization{
				Configurations: []v1alpha2.ConfigMapRef{},
			},
		}
	)

	BeforeEach(func() {
		namespace = createNamespace()

		configureAuthorizationToUnSecure(namespace.Name, userConfigurationConfigMapName)
		jenkins = createJenkinsCRSafeRestart(jenkinsCRName, namespace.Name, nil, groovyScripts, casc, priorityClassName)
	})

	AfterEach(func() {
		destroyNamespace(namespace)
	})

	Context("when running Jenkins safe restart", func() {
		It("authorization strategy is not overwritten", func() {
			waitForJenkinsBaseConfigurationToComplete(jenkins)
			waitForJenkinsUserConfigurationToComplete(jenkins)
			jenkinsClient, cleanUpFunc := verifyJenkinsAPIConnection(jenkins, namespace.Name)
			defer cleanUpFunc()
			checkIfAuthorizationStrategyUnsecuredIsSet(jenkinsClient)

			err := jenkinsClient.SafeRestart()
			Expect(err).NotTo(HaveOccurred())
			waitForJenkinsSafeRestart(jenkinsClient)

			checkIfAuthorizationStrategyUnsecuredIsSet(jenkinsClient)
		})
	})
})
