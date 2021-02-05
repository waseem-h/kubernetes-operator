package e2e

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	retryInterval = time.Second * 5
)

func waitForJenkinsBaseConfigurationToComplete(jenkins *v1alpha2.Jenkins) {
	By("waiting for Jenkins base configuration phase to complete")

	Eventually(func() (*metav1.Time, error) {
		actualJenkins := &v1alpha2.Jenkins{}
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, actualJenkins)
		if err != nil {
			return nil, err
		}

		return actualJenkins.Status.BaseConfigurationCompletedTime, nil
	}, time.Duration(170)*retryInterval, retryInterval).Should(Not(BeNil()))

	_, _ = fmt.Fprintf(GinkgoWriter, "Jenkins pod is running\n")

	// update jenkins CR because Operator sets default values
	namespacedName := types.NamespacedName{Namespace: jenkins.Namespace, Name: jenkins.Name}
	Expect(k8sClient.Get(context.TODO(), namespacedName, jenkins)).Should(Succeed())
}

func waitForRecreateJenkinsMasterPod(jenkins *v1alpha2.Jenkins) {
	By("waiting for Jenkins Master Pod recreation")

	Eventually(func() (bool, error) {
		lo := &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(resources.GetJenkinsMasterPodLabels(*jenkins)),
			Namespace:     jenkins.Namespace,
		}
		pods := &corev1.PodList{}
		err := k8sClient.List(context.TODO(), pods, lo)
		if err != nil {
			return false, err
		}
		if len(pods.Items) != 1 {
			return false, nil
		}

		return pods.Items[0].DeletionTimestamp == nil, nil
	}, 30*retryInterval, retryInterval).Should(BeTrue())

	_, _ = fmt.Fprintf(GinkgoWriter, "Jenkins pod has been recreated\n")
}

func waitForJenkinsUserConfigurationToComplete(jenkins *v1alpha2.Jenkins) {
	By("waiting for Jenkins user configuration phase to complete")

	Eventually(func() (*metav1.Time, error) {
		actualJenkins := &v1alpha2.Jenkins{}
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, actualJenkins)
		if err != nil {
			return nil, err
		}
		return actualJenkins.Status.UserConfigurationCompletedTime, nil
	}, time.Duration(110)*retryInterval, retryInterval).Should(Not(BeNil()))
	_, _ = fmt.Fprintf(GinkgoWriter, "Jenkins instance is up and ready\n")
}

func waitForJenkinsSafeRestart(jenkinsClient jenkinsclient.Jenkins) {
	By("waiting for Jenkins safe restart")

	Eventually(func() (bool, error) {
		status, err := jenkinsClient.Poll()
		_, _ = fmt.Fprintf(GinkgoWriter, "Safe restart status: %+v, err: %s\n", status, err)
		if err != nil {
			return false, err
		}
		if status != http.StatusOK {
			return false, err
		}
		return true, nil
	}, time.Second*200, time.Second*5).Should(BeTrue())
}
