package e2e

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/controllers"
	jenkinsClient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications"
	e "github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment

	hostname    *string
	port        *int
	useNodePort *bool
)

func init() {
	hostname = flag.String("jenkins-api-hostname", "", "Hostname or IP of Jenkins API. It can be service name, node IP or localhost.")
	port = flag.Int("jenkins-api-port", 0, "The port on which Jenkins API is running. Note: If you want to use nodePort don't set this setting and --jenkins-api-use-nodeport must be true.")
	useNodePort = flag.Bool("jenkins-api-use-nodeport", false, "Connect to Jenkins API using the service nodePort instead of service port. If you want to set this as true - don't set --jenkins-api-port.")
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(false)))

	By("bootstrapping test environment")
	useExistingCluster := true
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
		//BinaryAssetsDirectory: path.Join("..", "..", "testbin", "bin"),
		UseExistingCluster: &useExistingCluster,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = v1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	// setup manager
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	// setup controller
	clientSet, err := kubernetes.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())

	// setup events
	events, err := event.New(cfg, constants.OperatorName)
	Expect(err).NotTo(HaveOccurred())
	notificationEvents := make(chan e.Event)
	go notifications.Listen(notificationEvents, events, k8sClient)

	jenkinsAPIConnectionSettings := jenkinsClient.JenkinsAPIConnectionSettings{
		Hostname:    *hostname,
		UseNodePort: *useNodePort,
		Port:        *port,
	}

	err = (&controllers.JenkinsReconciler{
		Client:                       k8sManager.GetClient(),
		Scheme:                       k8sManager.GetScheme(),
		JenkinsAPIConnectionSettings: jenkinsAPIConnectionSettings,
		ClientSet:                    *clientSet,
		Config:                       *cfg,
		NotificationEvents:           &notificationEvents,
		KubernetesClusterDomain:      "cluster.local",
	}).SetupWithManager(k8sManager)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).NotTo(HaveOccurred())
	}()

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).NotTo(BeNil())
	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func createNamespace() *corev1.Namespace {
	By("creating temporary namespace")

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%d", time.Now().Unix()),
		},
	}
	Expect(k8sClient.Create(context.TODO(), namespace)).Should(Succeed())
	return namespace
}

func destroyNamespace(namespace *corev1.Namespace) {
	By("deleting temporary namespace")

	Expect(k8sClient.Delete(context.TODO(), namespace)).Should(Succeed())

	Eventually(func() (bool, error) {
		namespaces := &corev1.NamespaceList{}
		err := k8sClient.List(context.TODO(), namespaces)
		if err != nil {
			return false, err
		}

		exists := false
		for _, namespaceItem := range namespaces.Items {
			if namespaceItem.Name == namespace.Name {
				exists = true
				break
			}
		}

		return !exists, nil
	}, time.Second*120, time.Second).Should(BeTrue())
}
