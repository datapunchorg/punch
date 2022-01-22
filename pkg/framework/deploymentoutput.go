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

type DeploymentOutput interface {
	Steps() []string
	Output() map[string]DeploymentStepOutput
}

type DeploymentOutputImpl struct {
	steps []string
	output map[string]DeploymentStepOutput
}

func (t *DeploymentOutputImpl) Steps() []string {
	var result []string
	result = append(result, t.steps...)
	return result
}
func (t *DeploymentOutputImpl) Output() map[string]DeploymentStepOutput {
	result := map[string]DeploymentStepOutput{}
	for key, value := range t.output {
		result[key] = value
	}
	return result
}
