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

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
)

type DeploymentOutput interface {
	Steps() []string
	Output() map[string]DeployableOutput
}

type DeploymentOutputImpl struct {
	steps  []string
	output map[string]DeployableOutput
}

func (t *DeploymentOutputImpl) Steps() []string {
	var result []string
	result = append(result, t.steps...)
	return result
}
func (t *DeploymentOutputImpl) Output() map[string]DeployableOutput {
	result := map[string]DeployableOutput{}
	for key, value := range t.output {
		result[key] = value
	}
	return result
}

func MarshalDeploymentOutput(topologyKind string, deploymentOutput DeploymentOutput, jsonFormat bool) string {
	var output []DeploymentStepOutputStruct
	for _, name := range deploymentOutput.Steps() {
		output = append(output, DeploymentStepOutputStruct{
			Step:   name,
			Output: deploymentOutput.Output()[name],
		})
	}

	s := DeploymentOutputStruct{
		TopologyKind: topologyKind,
		Output: output,
	}

	if jsonFormat {
		jsonContent, err := json.Marshal(s)
		if err != nil {
			return fmt.Sprintf("<failed to marshal deployment output as JSON: %s>", err.Error())
		}
		return string(jsonContent)
	} else {
		yamlContent, err := yaml.Marshal(s)
		if err != nil {
			return fmt.Sprintf("<failed to marshal deployment output as YAML: %s>", err.Error())
		}
		return string(yamlContent)
	}
}

type DeploymentOutputStruct struct {
	TopologyKind string `json:"topologyKind" yaml:"topologyKind"`
	Output []DeploymentStepOutputStruct `json:"output" yaml:"output"`
}

type DeploymentStepOutputStruct struct {
	Step   string           `json:"step" yaml:"step"`
	Output DeployableOutput `json:"output" yaml:"output"`
}
