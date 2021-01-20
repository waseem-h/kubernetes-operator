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

package controllers

import (
	"context"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var _ = Describe("Jenkins controller", func() {
	Describe("deploying Jenkins CR into a cluster", func() {
		Context("when deploying CR to cluster", func() {
			It("create Jenkins instance", func() {
				ctx := context.Background()
				jenkins := createJenkinsCR(jenkinsCRName, namespace, &[]v1alpha2.SeedJob{mySeedJob.SeedJob}, groovyScripts, casc, priorityClassName)
				Expect(k8sClient.Create(ctx, jenkins)).Should(Succeed())
			})
		})
	})
})
