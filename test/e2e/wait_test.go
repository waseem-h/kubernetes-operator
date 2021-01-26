package e2e

import (
	"context"
	"fmt"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

/*func waitForRecreateJenkinsMasterPod(t *testing.T, jenkins *v1alpha2.Jenkins) {
	err := wait.Poll(retryInterval, 30*retryInterval, func() (bool, error) {
		lo := metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(resources.GetJenkinsMasterPodLabels(*jenkins)).String(),
		}
		podList, err := framework.Global.KubeClient.CoreV1().Pods(jenkins.ObjectMeta.Namespace).List(lo)
		if err != nil {
			return false, err
		}
		if len(podList.Items) != 1 {
			return false, nil
		}

		return podList.Items[0].DeletionTimestamp == nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fmt.Fprintf(GinkgoWriter,"Jenkins pod has been recreated")
}*/

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

/*func waitForJenkinsSafeRestart(t *testing.T, jenkinsClient jenkinsclient.Jenkins) {
	err := try.Until(func() (end bool, err error) {
		status, err := jenkinsClient.Poll()
		if err != nil {
			return false, err
		}
		if status != http.StatusOK {
			return false, errors.Wrap(err, "couldn't poll data from Jenkins API")
		}
		return true, nil
	}, time.Second, time.Second*70)
	require.NoError(t, err)
}*/
