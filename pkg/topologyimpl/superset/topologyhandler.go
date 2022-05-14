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

package superset

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"gopkg.in/yaml.v3"
	"log"
)

type TopologyHandler struct {
}

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	topology := GenerateSupersetTopology()
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	topology := GenerateSupersetTopology()
	err := yaml.Unmarshal(yamlContent, &topology)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML (%s): \n%s", err.Error(), string(yamlContent))
	}
	return &topology, nil
}

func (t *TopologyHandler) Validate(topology framework.Topology, phase string) (framework.Topology, error) {
	return topology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*SupersetTopology)
	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)
	deployment, err := CreateInstallDeployment(currentTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}
	err = deployment.Run()
	if err != nil {
		return deployment.GetOutput(), err
	}
	url := deployment.GetOutput().Output()["deploySupersetService"]["supersetUrl"].(string)
	log.Printf("Superset installed! Open %s in web browser to query data.", url)
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	return framework.NewDeployment().GetOutput(), nil
}

func (t *TopologyHandler) PrintUsageExample(topology framework.Topology, deploymentOutput framework.DeploymentOutput) {
	url := deploymentOutput.Output()["deploySupersetService"]["supersetUrl"].(string)
	log.Printf("Superset installed! Open %s in web browser to query data.", url)
}

func CreateInstallDeployment(topologySpec SupersetTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment := framework.NewDeployment()
	deployment.AddStep("deploySupersetService", "Deploy Superset Service", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		url, err := DeploySupersetService(commandEnvironment, topologySpec, topologySpec.Region, topologySpec.EksClusterName)
		if err != nil {
			return nil, err
		}
		return framework.DeployableOutput{
			"supersetUrl": url,
		}, nil
	})
	deployment.AddStep("addInitDatabases", "Add initial databases", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		url := c.GetStepOutput("deploySupersetService")["supersetUrl"].(string)
		err := AddInitDatabases(url, topologySpec)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	return deployment, nil
}

