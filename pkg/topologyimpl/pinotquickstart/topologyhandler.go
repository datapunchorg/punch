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

package pinotquickstart

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"gopkg.in/yaml.v3"
	"log"
)

type TopologyHandler struct {
}

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	topology := GeneratePinotQuickStartTopology()
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	topology := GeneratePinotQuickStartTopology()
	err := yaml.Unmarshal(yamlContent, &topology)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML (%s): \n%s", err.Error(), string(yamlContent))
	}
	return &topology, nil
}

func (t *TopologyHandler) Validate(topology framework.Topology, phase string) (framework.Topology, error) {
	currentTopology := topology.(*PinotQuickStartTopology)

	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)

	err := kubelib.CheckKubectl(commandEnvironment.Get(framework.CmdEnvKubectlExecutable))
	if err != nil {
		return nil, err
	}

	err = eks.ValidateEksTopologySpec(currentTopology.Spec.Eks, currentTopology.Metadata, phase)
	if err != nil {
		return nil, err
	}

	return topology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*PinotQuickStartTopology)
	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)
	deployment, err := CreateInstallDeployment(currentTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}
	err = deployment.Run()
	if err != nil {
		return deployment.GetOutput(), err
	}
	url := deployment.GetOutput().Output()["deployPinotService"]["pinotControllerUrl"].(string)
	log.Printf("Pinot installed! Open %s in web browser to query data.", url)
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*PinotQuickStartTopology)
	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)
	deployment, err := CreateUninstallDeployment(currentTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}
	err = deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) PrintNotes(topology framework.Topology, deploymentOutput framework.DeploymentOutput) {
	url := deploymentOutput.Output()["deployPinotService"]["pinotControllerUrl"].(string)
	log.Printf("Pinot installed! Open %s in web browser to query data.", url)
}

func CreateInstallDeployment(topologySpec PinotQuickStartTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment, err := eks.CreateInstallDeployment(topologySpec.Eks, commandEnvironment)
	if err != nil {
		return nil, err
	}
	deployment.AddStep("deployPinotService", "Deploy Pinot Service", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		url, err := DeployPinotService(commandEnvironment, topologySpec.Pinot, topologySpec.Eks.Region, topologySpec.Eks.EksCluster.ClusterName)
		if err != nil {
			return nil, err
		}
		return framework.DeployableOutput{
			"pinotControllerUrl": url,
		}, nil
	})
	deployment.AddStep("deployKafkaService", "Deploy Kafka Service", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		err := DeployKafkaService(commandEnvironment, topologySpec.Kafka, topologySpec.Eks.Region, topologySpec.Eks.EksCluster.ClusterName)
		return nil, err
	})
	deployment.AddStep("createKafkaTopics", "Create Kafka Topics", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		err := CreateKafkaTopics(commandEnvironment, topologySpec.Eks.Region, topologySpec.Eks.EksCluster.ClusterName)
		return nil, err
	})
	deployment.AddStep("createPinotRealtimeIngestion", "Create Pinot Realtime Ingestion", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		err := CreatePinotRealtimeIngestion(commandEnvironment, topologySpec.Eks.Region, topologySpec.Eks.EksCluster.ClusterName)
		return nil, err
	})
	return deployment, nil
}

func CreateUninstallDeployment(topologySpec PinotQuickStartTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment := framework.NewDeployment()
	deployment.AddStep("deletePinotLoadBalancer", "Delete Pinot Load Balancer", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		err := awslib.DeleteLoadBalancersOnEks(topologySpec.Eks.Region, topologySpec.Eks.VpcId, topologySpec.Eks.EksCluster.ClusterName, topologySpec.Pinot.Namespace)
		return framework.NewDeploymentStepOutput(), err
	})
	deployment2, err := eks.CreateUninstallDeployment(topologySpec.Eks, commandEnvironment)
	for _, step := range deployment2.GetSteps() {
		deployment.AddStep(step.GetName(), step.GetDescription(), step.GetDeployable())
	}
	return deployment, err
}
