# Jenkins Operator Roadmap

This document outlines the vision and technical roadmap for [jenkinsci/kubernetes-operator](https://github.com/jenkinsci/kubernetes-operator) project.

## Project Vision

With Jenkins Operator project we want to enable our community to run Jenkins in cloud-native environments like Kubernetes, OpenShift and others. Also, support most of the public cloud providers (AWS, Azure, GCP) in terms of additional capabilities like backups, observability and cloud security.  


With declarative configuration and full lifecycle management based on [Operator Framework](https://operatorframework.io/) this can become the de facto standard for running Jenkins on top of Kubernetes.


Now, we have a dedicated team which can bring more engineering effort to the project.

## Technical Roadmap

- Upgrade Operator SDK [#494](https://github.com/jenkinsci/kubernetes-operator/issues/494)
    - https://github.com/operator-framework/operator-sdk/releases/tag/v1.3.0
- Modularise codebase and define areas of responsibility (AWS, Azure, OpenShift)
- Break down Jenkins Custom Resource into smaller parts, support multiple Custom Resource Definitions [#495](https://github.com/jenkinsci/kubernetes-operator/issues/495)
    - Introduce more granular schema for configuring Jenkins
    - Introduce independent reconciliation controllers
    - Refactor e2e tests to support testing individual reconciliation controller
- Improve contribution process and establish governance model [#496](https://github.com/jenkinsci/kubernetes-operator/issues/496)
    - Improve CONTRIBUTING.md
    - Introduce architecture decision proposals process
    - Introduce governance model
- Migrate Jenkins instance from Pod to Deployment [#497](https://github.com/jenkinsci/kubernetes-operator/issues/497)
    - Unblock easier integration with 3rd party systems e.g. sidecars injection
- Reference Architecture - Jenkins on Kubernetes
    - Bridge the gap between Jenkins and Kubernetes
    - https://www.jenkins.io/blog/2020/12/04/gsod-project-report/
- Pay technical debt
    - Review existing issues on GitHub and prioritise them
    - Introduce project board to track milestones
    - Fix existing bugs

## Have a Question?

In case of further question feel free to create an issue or contact us directly.
