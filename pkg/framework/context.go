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

type DeploymentContext interface {
	AddStepOutput(name string, output map[string]interface{})
	GetStepOutput(name string) map[string]interface{}
}

type DefaultDeploymentContext struct {
	outputs map[string]map[string]interface{}
}

func NewDefaultDeploymentContext() DeploymentContext {
	return DefaultDeploymentContext{
		outputs: map[string]map[string]interface{}{},
	}
}

func (d DefaultDeploymentContext) AddStepOutput(name string, output map[string]interface{}) {
	d.outputs[name] = output
}

func (d DefaultDeploymentContext) GetStepOutput(name string) map[string]interface{} {
	output := d.outputs[name]
	return output
}
