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
	"log"
)

type Deployment interface {
	GetContext() DeploymentContext
	AddStep(name string, description string, run DeploymentStepFunc)
	Run() error
	GetOutput() DeploymentOutput
}

type deploymentImpl struct {
	context DeploymentContext
	steps   []DeploymentStep
}

func NewDeployment() Deployment {
	return &deploymentImpl{
		context: NewDeploymentContext(),
		steps:   []DeploymentStep{},
	}
}

func (d *deploymentImpl) GetContext() DeploymentContext {
	return d.context
}

func (d *deploymentImpl) AddStep(name string, description string, run DeploymentStepFunc) {
	step := deploymentStepWrapper{
		name:        name,
		description: description,
		run:         run,
	}
	d.steps = append(d.steps, step)
}

func (d *deploymentImpl) Run() error {
	for _, step := range d.steps {
		log.Printf("[StepBegin] %s: %s", step.Name(), step.Description())
		output, err := step.Run(d.context)
		if err != nil {
			return fmt.Errorf("failed to run step %s: %s", step.Name(), err.Error())
		}
		d.context.AddStepOutput(step.Name(), output)
		log.Printf("[StepEnd] %s: %s", step.Name(), step.Description())
	}
	return nil
}

func (d *deploymentImpl) GetOutput() DeploymentOutput {
	result := DeploymentOutputImpl{
		steps:  []string{},
		output: map[string]DeploymentStepOutput{},
	}
	names := d.getStepNames()
	for _, name := range names {
		result.steps = append(result.steps, name)
		result.output[name] = d.GetContext().GetStepOutput(name)
	}
	return &result
}

func (d *deploymentImpl) getStepNames() []string {
	var result []string
	for _, entry := range d.steps {
		result = append(result, entry.Name())
	}
	return result
}
