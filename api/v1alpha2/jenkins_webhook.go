/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha2

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/plugins"

	"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var jenkinslog = logf.Log.WithName("jenkins-resource")

func (in *Jenkins) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-jenkins-io-jenkins-io-v1alpha2-jenkins,mutating=false,failurePolicy=fail,sideEffects=None,groups=jenkins.io.jenkins.io,resources=jenkins,verbs=create;update,versions=v1alpha2,name=vjenkins.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Jenkins{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Jenkins) ValidateCreate() error {
	if in.Spec.ValidateSecurityWarnings {
		jenkinslog.Info("validate create", "name", in.Name)
		return Validate(*in)
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Jenkins) ValidateUpdate(old runtime.Object) error {
	if in.Spec.ValidateSecurityWarnings {
		jenkinslog.Info("validate update", "name", in.Name)
		return Validate(*in)
	}

	return nil
}

func (in *Jenkins) ValidateDelete() error {
	return nil
}

type Warnings struct {
	Warnings []Warning `json:"securityWarnings"`
}

type Warning struct {
	Versions []Version `json:"versions"`
	ID       string    `json:"id"`
	Message  string    `json:"message"`
	URL      string    `json:"url"`
	Active   bool      `json:"active"`
}
type Version struct {
	FirstVersion string `json:"firstVersion"`
	LastVersion  string `json:"lastVersion"`
}

const APIURL string = "https://plugins.jenkins.io/api/plugin/"

func MakeSemanticVersion(version string) string {
	version = "v" + version
	return semver.Canonical(version)
}

func CompareVersions(firstVersion string, lastVersion string, pluginVersion string) bool {
	firstSemVer := MakeSemanticVersion(firstVersion)
	lastSemVer := MakeSemanticVersion(lastVersion)
	pluginSemVer := MakeSemanticVersion(pluginVersion)
	if semver.Compare(pluginSemVer, firstSemVer) == -1 || semver.Compare(pluginSemVer, lastSemVer) == 1 {
		return false
	}
	return true
}

func CheckSecurityWarnings(pluginName string, pluginVersion string) (bool, error) {
	jenkinslog.Info("checking security warnings", "plugin: ", pluginName)
	pluginURL := APIURL + pluginName
	client := &http.Client{
		Timeout: time.Second * 30,
	}
	request, err := http.NewRequest("GET", pluginURL, nil)
	if err != nil {
		return false, err
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return false, err
	}
	securityWarnings := Warnings{}

	jsonErr := json.Unmarshal(bodyBytes, &securityWarnings)
	if jsonErr != nil {
		return false, err
	}

	jenkinslog.Info("Validate()", "warnings", securityWarnings)

	for _, warning := range securityWarnings.Warnings {
		for _, version := range warning.Versions {
			firstVersion := version.FirstVersion
			lastVersion := version.LastVersion
			if len(firstVersion) == 0 {
				firstVersion = "0" // setting default value in case of empty string
			}
			if len(lastVersion) == 0 {
				lastVersion = pluginVersion // setting default value in case of empty string
			}

			if CompareVersions(firstVersion, lastVersion, pluginVersion) {
				jenkinslog.Info("security Vulnerabilities detected", "message", warning.Message, "Check security Advisory", warning.URL)
				return true, nil
			}
		}
	}

	return false, nil
}

func Validate(r Jenkins) error {
	basePlugins := plugins.BasePlugins()
	var warnings string = ""

	for _, plugin := range basePlugins {
		name := plugin.Name
		version := plugin.Version
		hasWarnings, err := CheckSecurityWarnings(name, version)
		if err != nil {
			return err
		}
		if hasWarnings {
			warnings += "Security Vulnerabilities detected in base plugin:" + name
		}
	}

	for _, plugin := range r.Spec.Master.Plugins {
		name := plugin.Name
		version := plugin.Version
		hasWarnings, err := CheckSecurityWarnings(name, version)
		if err != nil {
			return err
		}
		if hasWarnings {
			warnings += "Security Vulnerabilities detected in the user defined plugin: " + name
		}
	}
	if len(warnings) > 0 {
		return errors.New(warnings)
	}

	return nil
}
