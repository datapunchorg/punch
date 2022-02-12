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

const (
	PunchTopologyTagName = "punch-topology"

	DefaultEKSAssumeRolePolicyDocument = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":["eks.amazonaws.com"]},"Action":["sts:AssumeRole"]}]}`
	DefaultEC2AssumeRolePolicyDocument = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":["ec2.amazonaws.com"]},"Action":["sts:AssumeRole"]}]}`
)

type Topology interface {
	GetKind() string
	// TODO remove ToString and make password mask generic function, instead of doing password mask inside ToString
	ToString() string
}

type TopologyMetadata struct {
	Name               string            `json:"name"`
	CommandEnvironment map[string]string `json:"commandEnvironment" yaml:"commandEnvironment"`
	Notes              map[string]string `json:"notes" yaml:"notes"`
}
