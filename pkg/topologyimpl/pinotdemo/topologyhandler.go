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

package pinotdemo

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"gopkg.in/yaml.v3"
	"log"
)

const (
	sparkcliJavaExampleCommandFormat   = `./sparkcli --user %s --password %s --insecure --url %s/sparkapi/v1 submit --class org.apache.spark.examples.SparkPi --spark-version 3.2 --driver-memory 512m --executor-memory 512m local:///opt/spark/examples/jars/spark-examples_2.12-3.2.1.jar`
	sparkcliPythonExampleCommandFormat = `./sparkcli --user %s --password %s --insecure --url %s/sparkapi/v1 submit --spark-version 3.2 --driver-memory 512m --executor-memory 512m %s`
)

type TopologyHandler struct {
}

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	topology := GeneratePinotDemoTopology()
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	topology := GeneratePinotDemoTopology()
	err := yaml.Unmarshal(yamlContent, &topology)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML (%s): \n%s", err.Error(), string(yamlContent))
	}
	return &topology, nil
}

func (t *TopologyHandler) Validate(topology framework.Topology, phase string) (framework.Topology, error) {
	currentTopology := topology.(*PinotDemoTopology)

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
	currentTopology := topology.(*PinotDemoTopology)
	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)
	deployment, err := CreateInstallDeployment(currentTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}
	err = deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*PinotDemoTopology)
	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)
	deployment, err := CreateUninstallDeployment(currentTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}
	err = deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) PrintUsageExample(topology framework.Topology, deploymentOutput framework.DeploymentOutput) {
	log.Printf("Pinot Installed")
}

func CreateInstallDeployment(topologySpec PinotDemoTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment, err := eks.CreateInstallDeployment(topologySpec.Eks, commandEnvironment)
	if err != nil {
		return nil, err
	}
	deployment.AddStep("deployPinotServer", "Deploy Pinot Server", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		DeployPinotServer(commandEnvironment, topologySpec.Pinot, topologySpec.Eks.Region, topologySpec.Eks.EksCluster.ClusterName)
		return nil, nil
	})
	return deployment, nil
}

func CreateUninstallDeployment(topologySpec PinotDemoTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
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
