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
	jenkinslog                           = logf.Log.WithName("jenkins-resource") // log is for logging in this package.
	PluginsDataManager PluginDataManager = *NewPluginsDataManager()
	_                  webhook.Validator = &Jenkins{}
)

func (in *Jenkins) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-jenkins-io-jenkins-io-v1alpha2-jenkins,mutating=false,failurePolicy=fail,sideEffects=None,groups=jenkins.io.jenkins.io,resources=jenkins,verbs=create;update,versions=v1alpha2,name=vjenkins.kb.io,admissionReviewVersions={v1,v1beta1}

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

type PluginDataManager struct {
	pluginDataCache    PluginsInfo
	hosturl            string
	compressedFilePath string
	pluginDataFile     string
	iscached           bool
	maxattempts        int
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
	var faultybaseplugins string
	var faultyuserplugins string
	basePlugins := plugins.BasePlugins()

	for _, plugin := range basePlugins {
		// Only Update the map if the plugin is not present or a lower version is being used
		if pluginData, ispresent := pluginset[plugin.Name]; !ispresent || semver.Compare(MakeSemanticVersion(plugin.Version), pluginData.Version) == 1 {
			pluginset[plugin.Name] = PluginData{Version: plugin.Version, Kind: "base"}
		}
	}

	for _, plugin := range r.Spec.Master.Plugins {
		if pluginData, ispresent := pluginset[plugin.Name]; !ispresent || semver.Compare(MakeSemanticVersion(plugin.Version), pluginData.Version) == 1 {
			pluginset[plugin.Name] = PluginData{Version: plugin.Version, Kind: "user-defined"}
		}
	}

	for _, plugin := range PluginsDataManager.pluginDataCache.Plugins {
		if pluginData, ispresent := pluginset[plugin.Name]; ispresent {
			var hasvulnerabilities bool
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
						jenkinslog.Info("Security Vulnerability detected in "+pluginData.Kind+" "+plugin.Name+":"+pluginData.Version, "Warning message", warning.Message, "For more details,check security advisory", warning.URL)
						hasvulnerabilities = true
					}
				}
			}

			if hasvulnerabilities {
				if pluginData.Kind == "base" {
					faultybaseplugins += plugin.Name + ":" + pluginData.Version + "\n"
				} else {
					faultyuserplugins += plugin.Name + ":" + pluginData.Version + "\n"
				}
			}
		}
	}
	if len(faultybaseplugins) > 0 || len(faultyuserplugins) > 0 {
		var errormsg string
		if len(faultybaseplugins) > 0 {
			errormsg += "Security vulnerabilities detected in the following base plugins: \n" + faultybaseplugins
		}
		if len(faultyuserplugins) > 0 {
			errormsg += "Security vulnerabilities detected in the following user-defined plugins: \n" + faultyuserplugins
		}
		return errors.New(errormsg)
	}

	return nil
}

func NewPluginsDataManager() *PluginDataManager {
	return &PluginDataManager{
		hosturl:            "https://ci.jenkins.io/job/Infra/job/plugin-site-api/job/generate-data/lastSuccessfulBuild/artifact/plugins.json.gzip",
		compressedFilePath: "/tmp/plugins.json.gzip",
		pluginDataFile:     "/tmp/plugins.json",
		iscached:           false,
		maxattempts:        5,
	}
}

// Downloads extracts and caches the JSON data in every 12 hours
func (in *PluginDataManager) CachePluginData(ch chan bool) {
	for {
		jenkinslog.Info("Initializing/Updating the plugin data cache")
		var isdownloaded, isextracted, iscached bool
		var err error
		for i := 0; i < in.maxattempts; i++ {
			err = in.Download()
			if err == nil {
				isdownloaded = true
				break
			}
		}

		if isdownloaded {
			for i := 0; i < in.maxattempts; i++ {
				err = in.Extract()
				if err == nil {
					isextracted = true
					break
				}
			}
		} else {
			jenkinslog.Info("Cache Plugin Data", "failed to download file", err)
		}

		if isextracted {
			for i := 0; i < in.maxattempts; i++ {
				err = in.Cache()
				if err == nil {
					iscached = true
					break
				}
			}

			if !iscached {
				jenkinslog.Info("Cache Plugin Data", "failed to read plugin data file", err)
			}
		} else {
			jenkinslog.Info("Cache Plugin Data", "failed to extract file", err)
		}

		if !in.iscached {
			ch <- iscached
		}
		in.iscached = in.iscached || iscached
		time.Sleep(12 * time.Hour)
	}
}

func (in *PluginDataManager) Download() error {
	out, err := os.Create(in.compressedFilePath)
	if err != nil {
		return err
	}
	defer out.Close()

	client := http.Client{
		Timeout: 1000 * time.Second,
	}

	resp, err := client.Get(in.hosturl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (in *PluginDataManager) Extract() error {
	reader, err := os.Open(in.compressedFilePath)

	if err != nil {
		return err
	}
	defer reader.Close()
	archive, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}

	defer archive.Close()
	writer, err := os.Create(in.pluginDataFile)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, archive)
	return err
}

// Loads the JSON data into memory and stores it
func (in *PluginDataManager) Cache() error {
	jsonFile, err := os.Open(in.pluginDataFile)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(byteValue, &in.pluginDataCache)
	if err != nil {
		return err
	}
	return nil
}

// returns a semantic version that can be used for comparison
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
