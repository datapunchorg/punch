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
	AddStep(name string, description string, run DeployableFunction)
	GetSteps() []DeploymentStep
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

func (d *deploymentImpl) AddStep(name string, description string, run DeployableFunction) {
	step := deploymentStepStruct{
		name:        name,
		description: description,
		run:         run,
	}
	d.steps = append(d.steps, step)
}

func (d *deploymentImpl) GetSteps() []DeploymentStep {
	return d.steps
}

func (d *deploymentImpl) Run() error {
	for _, step := range d.steps {
		log.Printf("[StepBegin] %s: %s", step.GetName(), step.GetDescription())
		deployable := step.GetDeployable()
		output, err := deployable(d.context)
		if err != nil {
			return fmt.Errorf("failed to run step %s: %s", step.GetName(), err.Error())
		}
		d.context.AddStepOutput(step.GetName(), output)
		log.Printf("[StepEnd] %s: %s", step.GetName(), step.GetDescription())
	}
	return nil
}

func (d *deploymentImpl) GetOutput() DeploymentOutput {
	result := DeploymentOutputImpl{
		steps:  []string{},
		output: map[string]DeployableOutput{},
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
		result = append(result, entry.GetName())
	}
	return result
}
