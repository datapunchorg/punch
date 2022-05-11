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

package kyuubioneks

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"github.com/datapunchorg/punch/pkg/topologyimpl/sparkoneks"
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
	topology := GenerateKyuubiOnEksTopology()
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	topology := GenerateKyuubiOnEksTopology()
	err := yaml.Unmarshal(yamlContent, &topology)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML (%s): \n%s", err.Error(), string(yamlContent))
	}
	return &topology, nil
}

func (t *TopologyHandler) Validate(topology framework.Topology, phase string) (framework.Topology, error) {
	currentTopology := topology.(*KyuubiOnEksTopology)

	err := sparkoneks.ValidateSparkComponentSpec(currentTopology.Spec.Spark, currentTopology.Metadata, phase)
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
	currentTopology := topology.(*KyuubiOnEksTopology)
	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)
	deployment, err := CreateInstallDeployment(currentTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}
	err = deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*KyuubiOnEksTopology)
	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)
	deployment, err := CreateUninstallDeployment(currentTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}
	err = deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) PrintUsageExample(topology framework.Topology, deploymentOutput framework.DeploymentOutput) {
	log.Printf("Kyuubi Installed")
}

func CreateInstallDeployment(topologySpec KyuubiOnEksTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment, err := eks.CreateInstallDeployment(topologySpec.Eks, commandEnvironment)
	if err != nil {
		return nil, err
	}
	deployment.AddStep("deploySparkOperator", "Deploy Spark Operator", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		err := sparkoneks.DeploySparkOperator(commandEnvironment, topologySpec.Spark, topologySpec.Eks.Region, topologySpec.Eks.EksCluster.ClusterName, topologySpec.Eks.S3BucketName)
		return nil, err
	})
	deployment.AddStep("deploySparkHistoryServer", "Deploy Spark History Server", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		err := sparkoneks.DeployHistoryServer(commandEnvironment, topologySpec.Spark, topologySpec.Eks.Region, topologySpec.Eks.EksCluster.ClusterName)
		return nil, err
	})
	return deployment, nil
}

func CreateUninstallDeployment(topologySpec KyuubiOnEksTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment, err := eks.CreateUninstallDeployment(topologySpec.Eks, commandEnvironment)
	return deployment, err
}
