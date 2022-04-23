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

package kafka

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"gopkg.in/yaml.v3"
	"log"
	"regexp"
)

var nonAlphanumericRegexp *regexp.Regexp

func init() {
	framework.DefaultTopologyHandlerManager.AddHandler(KindKafkaTopology, &TopologyHandler{})

	var err error
	nonAlphanumericRegexp, err = regexp.Compile("[^a-zA-Z]+")
	if err != nil {
		panic(err)
	}
}

type TopologyHandler struct {
}

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	namePrefix := "{{ or .Values.namePrefix `my` }}"
	topology := CreateDefaultKafkaTopology(namePrefix)
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	result := CreateDefaultKafkaTopology(DefaultNamePrefix)
	err := yaml.Unmarshal(yamlContent, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML (%s): \n%s", err.Error(), string(yamlContent))
	}
	return &result, nil
}

func (t *TopologyHandler) Validate(topology framework.Topology, install bool) (framework.Topology, error) {
	return topology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	kafkaTopology := topology.(*KafkaTopology)
	deployment := framework.NewDeployment()
	deployment.AddStep("createKafkaCluster", "Create Kafka cluster", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
		cluster, err := CreateKafkaCluster(kafkaTopology.Spec)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		bootstrap, err := GetBootstrapBrokerString(kafkaTopology.Spec.Region, *cluster.ClusterArn)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		return framework.DeploymentStepOutput{
			"kafkaClusterArn": cluster.ClusterArn,
			"bootstrapServerString": *bootstrap.BootstrapBrokerStringSaslIam,
		}, nil
	})
	err := deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	kafkaTopology := topology.(*KafkaTopology)
	deployment := framework.NewDeployment()
	deployment.AddStep("deleteKafkaCluster", "Delete Kafka cluster", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
		err := DeleteKafkaCluster(kafkaTopology.Spec.Region, kafkaTopology.Spec.ClusterName)
		if err != nil {
			log.Printf("[WARN] Cannot delete Kafka cluster %s in region %s: %s", kafkaTopology.Spec.ClusterName, kafkaTopology.Spec.Region, err.Error())
		}
		return framework.NewDeploymentStepOutput(), nil
	})
	err := deployment.Run()
	return deployment.GetOutput(), err
}

