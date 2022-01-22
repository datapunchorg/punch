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

package main

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"gopkg.in/yaml.v3"
)

func MarshalDeploymentOutput(deploymentOutput framework.DeploymentOutput) string {
	var output []DeploymentStepOutputMarshalStruct
	for _, name := range deploymentOutput.Steps() {
		output = append(output, DeploymentStepOutputMarshalStruct{
			Step:   name,
			Output: deploymentOutput.Output()[name],
		})
	}

	s := DeploymentOutputMarshalStruct{
		Output: output,
	}

	yamlContent, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Sprintf("<failed to marshal deployment output: %s>", err.Error())
	}
	return string(yamlContent)
}

type DeploymentOutputMarshalStruct struct {
	Output []DeploymentStepOutputMarshalStruct `json:"output" yaml:"output"`
}

type DeploymentStepOutputMarshalStruct struct {
	Step   string                         `json:"step" yaml:"step"`
	Output framework.DeploymentStepOutput `json:"output" yaml:"output"`
}