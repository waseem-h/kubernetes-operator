package controllers

import (
	"context"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var logxx = log.Log

// JenkinsImageReconciler reconciles a JenkinsImage object
type JenkinsImageReconciler struct {
	// This Client, initialized using mgr.Client() above, is a split Client
	// that reads objects from the cache and writes to the apiserver
	Client client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *JenkinsImageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.JenkinsImage{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// Reconcile reads that state of the cluster for a JenkinsImage object and makes changes based on the state read
// and what is in the JenkinsImage.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *JenkinsImageReconciler) Reconcile(_ context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := logxx.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling JenkinsImage")

	// Fetch the JenkinsImage instance
	instance := &v1alpha2.JenkinsImage{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new ConfigMap containing the Dockerfile used to build the image
	dockerfile := resources.NewDockerfileConfigMap(instance)
	// Set JenkinsImage instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, dockerfile, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this ConfigMap already exists
	foundConfigMap := &corev1.ConfigMap{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: dockerfile.Name, Namespace: dockerfile.Namespace}, foundConfigMap)
	if err != nil && apierrors.IsNotFound(err) {
		reqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", dockerfile.Namespace, "ConfigMap.Name", dockerfile.Name)
		err = r.Client.Create(context.TODO(), dockerfile)
		if err != nil {
			return reconcile.Result{}, err
		}
		// ConfigMap created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}
	// ConfigMap already exists - don't requeue
	reqLogger.Info("Skip reconcile: ConfigMap already exists", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)

	// Define a new Pod object
	pod := resources.NewBuilderPod(instance)
	// Set JenkinsImage instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pod, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	foundPod := &corev1.Pod{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, foundPod)
	if err != nil && apierrors.IsNotFound(err) {
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.Client.Create(context.TODO(), pod)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}
	// Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", foundPod.Namespace, "Pod.Name", foundPod.Name)

	return reconcile.Result{}, nil
}
