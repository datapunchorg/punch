/*
Copyright 2022 DataPunch Organization

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

type DeploymentStep interface {
	GetName() string
	GetDescription() string
	GetDeployable() DeployableFunction
}

type deploymentStepStruct struct {
	name        string
	description string
	run         DeployableFunction
}

func (d deploymentStepStruct) GetName() string {
	return d.name
}

func (d deploymentStepStruct) GetDescription() string {
	return d.description
}

func (d deploymentStepStruct) GetDeployable() DeployableFunction {
	return d.run
}

type DeployableFunction func(context DeploymentContext) (DeployableOutput, error)

type DeployableOutput map[string]interface{}

func NewDeploymentStepOutput() DeployableOutput {
	return DeployableOutput{}
}
