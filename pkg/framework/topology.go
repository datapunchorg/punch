/*
Copyright 2022 DataPunch Project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

const (
	DefaultVersion    = "datapunch.org/v1alpha1"

	CmdEnvHelmExecutable             = "helmExecutable"
	CmdEnvWithMinikube               = "withMinikube"
	CmdEnvKubeConfig                 = "kubeConfig"

	DefaultRegion         = "us-west-1"
	DefaultNamePrefix     = "my"

	DefaultHelmExecutable = "helm"

	DefaultEKSAssumeRolePolicyDocument = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":["eks.amazonaws.com"]},"Action":["sts:AssumeRole"]}]}`
	DefaultEC2AssumeRolePolicyDocument = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":["ec2.amazonaws.com"]},"Action":["sts:AssumeRole"]}]}`
)

type TopologyBase struct {
	ApiVersion string                     `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                     `json:"kind" yaml:"kind"`
	Metadata   TopologyMetadata 		  `json:"metadata"`
}

type Topology interface {
	GetKind() string
	GetMetadata() *TopologyMetadata
}

type TopologyMetadata struct {
	Name               string            `json:"name"`
	CommandEnvironment map[string]string `json:"commandEnvironment" yaml:"commandEnvironment"`
	Notes              map[string]string `json:"notes" yaml:"notes"`
}

func TopologyString(topology Topology) string {
	s, ok := topology.(fmt.Stringer)
	if ok {
		return s.String()
	} else {
		topologyBytes, err := yaml.Marshal(topology)
		if err != nil {
			return fmt.Sprintf("(Failed to serialize topology: %s)", err.Error())
		}
		return string(topologyBytes)
	}
}