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

type DeploymentStep interface {
	Name() string
	Description() string
	Run(context DeploymentContext) (DeploymentStepOutput, error)
}

type deploymentStepWrapper struct {
	name        string
	description string
	run         DeploymentStepFunc
}

func (d deploymentStepWrapper) Name() string {
	return d.name
}

func (d deploymentStepWrapper) Description() string {
	return d.description
}

func (d deploymentStepWrapper) Run(context DeploymentContext) (DeploymentStepOutput, error) {
	return d.run(context)
}

type DeploymentStepFunc func(context DeploymentContext) (DeploymentStepOutput, error)

type DeploymentStepOutput map[string]interface{}

func NewDeploymentStepOutput() DeploymentStepOutput {
	return DeploymentStepOutput{}
}
