/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/lictenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha2

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/plugins"

	"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	jenkinslog       = logf.Log.WithName("jenkins-resource") // log is for logging in this package.
	isRetrieved bool = false                                 // For checking whether the data file is downloaded and extracted or not
)

const (
	hosturl        string = "https://ci.jenkins.io/job/Infra/job/plugin-site-api/job/generate-data/lastSuccessfulBuild/artifact/plugins.json.gzip" // Url for downloading the plugins file
	compressedFile string = "/tmp/plugins.json.gzip"                                                                                               // location where the gzip file will be downloaded
	pluginDataFile string = "/tmp/plugins.json"                                                                                                    // location where the file will be extracted
)

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

type PluginsInfo struct {
	Plugins []PluginInfo `json:"plugins"`
}

type PluginInfo struct {
	Name             string    `json:"name"`
	SecurityWarnings []Warning `json:"securityWarnings"`
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

type PluginData struct {
	Version string
	Kind    string
}

// Validates security warnings for both updating and creating a Jenkins CR
func Validate(r Jenkins) error {

	pluginset := make(map[string]PluginData)
	var warningmsg string
	basePlugins := plugins.BasePlugins()
	temp, err := NewPluginsInfo()
	AllPluginData := *temp
	if err != nil {
		return err
	}

	for _, plugin := range basePlugins {
		// Only Update the map if the plugin is not present or a lower version is being used
		if pluginData, ispresent := pluginset[plugin.Name]; !ispresent || semver.Compare(MakeSemanticVersion(plugin.Version), pluginData.Version) == 1 {
			jenkinslog.Info("Validate", plugin.Name, plugin.Version)
			pluginset[plugin.Name] = PluginData{Version: plugin.Version, Kind: "base"}
		}
	}

	for _, plugin := range r.Spec.Master.Plugins {
		if pluginData, ispresent := pluginset[plugin.Name]; !ispresent || semver.Compare(MakeSemanticVersion(plugin.Version), pluginData.Version) == 1 {
			jenkinslog.Info("Validate", plugin.Name, plugin.Version)
			pluginset[plugin.Name] = PluginData{Version: plugin.Version, Kind: "user-defined"}
		}
	}

	jenkinslog.Info("Checking through all the warnings")
	for _, plugin := range AllPluginData.Plugins {

		if pluginData, ispresent := pluginset[plugin.Name]; ispresent {
			jenkinslog.Info("Checking for plugin", "name", plugin.Name)
			for _, warning := range plugin.SecurityWarnings {
				for _, version := range warning.Versions {
					firstVersion := version.FirstVersion
					lastVersion := version.LastVersion
					if len(firstVersion) == 0 {
						firstVersion = "0" // setting default value in case of empty string
					}
					if len(lastVersion) == 0 {
						lastVersion = pluginData.Version // setting default value in case of empty string
					}

					if CompareVersions(firstVersion, lastVersion, pluginData.Version) {
						jenkinslog.Info("security Vulnerabilities detected", "message", warning.Message, "Check security Advisory", warning.URL)
						warningmsg += "Security Vulnerabilities detected in " + pluginData.Kind + " plugin " + plugin.Name + "\n"

					}

				}
			}

		}

	}

	if len(warningmsg) > 0 {
		return errors.New(warningmsg)
	}

	return nil

}

// Returns an object containing information of all the plugins present in the security center
func NewPluginsInfo() (*PluginsInfo, error) {
	var AllPluginData PluginsInfo
	for i := 0; i < 28; i++ {
		if isRetrieved {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !isRetrieved {
		jenkinslog.Info("Plugins Data file hasn't been downloaded and extracted")
		return &AllPluginData, errors.New("plugins data file not found")
	}

	jsonFile, err := os.Open(pluginDataFile)
	if err != nil {
		jenkinslog.Info("Failed to open the Plugins Data File")
		return &AllPluginData, err
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		jenkinslog.Info("Failed to convert the JSON file into a byte array")
		return &AllPluginData, err
	}
	err = json.Unmarshal(byteValue, &AllPluginData)
	if err != nil {
		jenkinslog.Info("Failed to decode the Plugin JSON data file")
		return &AllPluginData, err
	}

	return &AllPluginData, nil
}

// Downloads and extracts the JSON file in every 12 hours
func RetrieveDataFile() {
	for {
		jenkinslog.Info("Retreiving file", "Host Url", hosturl)
		err := Download()
		if err != nil {
			jenkinslog.Info("Retrieving File", "Error while downloading", err)
			continue
		}

		jenkinslog.Info("Retrieve File", "Successfully downloaded", compressedFile)
		err = Extract()
		if err != nil {
			jenkinslog.Info("Retreive File", "Error while extracting", err)
			continue
		}
		jenkinslog.Info("Retreive File", "Successfully extracted", pluginDataFile)
		isRetrieved = true
		time.Sleep(12 * time.Hour)

	}
}

func Download() error {

	out, err := os.Create(compressedFile)
	if err != nil {
		return err
	}
	defer out.Close()

	client := http.Client{
		Timeout: 2000 * time.Second,
	}

	resp, err := client.Get(hosturl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	jenkinslog.Info("Successfully Downloaded")
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil

}

func Extract() error {
	reader, err := os.Open(compressedFile)

	if err != nil {
		return err
	}
	defer reader.Close()
	archive, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}

	defer archive.Close()
	writer, err := os.Create(pluginDataFile)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, archive)
	return err

}

// returns a semantic version that can be used for comparision
func MakeSemanticVersion(version string) string {
	version = "v" + version
	return semver.Canonical(version)
}

// Compare if the current version lies between first version and last version
func CompareVersions(firstVersion string, lastVersion string, pluginVersion string) bool {
	firstSemVer := MakeSemanticVersion(firstVersion)
	lastSemVer := MakeSemanticVersion(lastVersion)
	pluginSemVer := MakeSemanticVersion(pluginVersion)
	if semver.Compare(pluginSemVer, firstSemVer) == -1 || semver.Compare(pluginSemVer, lastSemVer) == 1 {
		return false
	}
	return true
}
