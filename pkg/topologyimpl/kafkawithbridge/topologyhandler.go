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

package kafkawithbridge

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/common"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"github.com/datapunchorg/punch/pkg/topologyimpl/kafkaonmsk"
	"gopkg.in/yaml.v3"
	"log"
	"regexp"
	"time"
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

type createTopicRequest struct {
	NumPartitions int64  `json:"numPartitions" yaml:"numPartitions"`
	ReplicationFactor int64  `json:"replicationFactor" yaml:"replicationFactor"`
}

type createTopicResponse struct {
	Name string `json:"name" yaml:"name"`
}

type TopologyHandler struct {
}

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	topology := GenerateDefaultTopology()
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	topology := GenerateDefaultTopology()
	err := yaml.Unmarshal(yamlContent, &topology)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML (%s): \n%s", err.Error(), string(yamlContent))
	}
	return &topology, nil
}

func (t *TopologyHandler) Validate(topology framework.Topology, phase string) (framework.Topology, error) {
	// TODO
	return topology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*KafkaWithBridgeTopology)
	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)
	deployment := kafkaonmsk.CreateInstallDeployment(currentTopology.Spec.KafkaOnMskSpec)
	deployment2, err := eks.CreateInstallDeployment(currentTopology.Spec.EksSpec, commandEnvironment)
	if err != nil {
		return nil, err
	}
	for _, step := range deployment2.GetSteps() {
		deployment.AddStep(step.GetName(), step.GetDescription(), step.GetDeployable())
	}
	deployment.AddStep("deployStrimziKafkaBridge", "Deploy Strimzi Kafka Bridge", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		spec := currentTopology.Spec
		if spec.KafkaBridge.KafkaBootstrapServers == "" {
			bootstrapServerString := c.GetStepOutput("createKafkaCluster")["bootstrapServerString"].(string)
			spec.KafkaBridge.KafkaBootstrapServers = bootstrapServerString
			log.Printf("Set spec.KafkaBridge.KafkaBootstrapServers: %s", bootstrapServerString)
		}
		loadBalancerUrl := c.GetStepOutput("deployNginxIngressController")["loadBalancerPreferredUrl"].(string)
		if loadBalancerUrl == "" {
			return framework.NewDeploymentStepOutput(), fmt.Errorf("did not find load balancer url")
		}
		DeployKafkaBridge(commandEnvironment, spec)
		kafkaBridgeUrl := fmt.Sprintf("%s/topics", loadBalancerUrl)
		maxWaitMinutes := 10
		err := common.WaitHttpUrlReady(kafkaBridgeUrl, time.Duration(maxWaitMinutes) * time.Minute, 10 * time.Second)
		if err != nil {
			return framework.NewDeploymentStepOutput(), fmt.Errorf("kafka bridge url %s is not ready after waiting %d minutes", kafkaBridgeUrl, maxWaitMinutes)
		}
		kafkaBridgeTopicProduceUrl := fmt.Sprintf("%s/topics", loadBalancerUrl)
		kafkaBridgeTopicAdminUrl := fmt.Sprintf("%s/topicAdmin", loadBalancerUrl)
		return framework.DeployableOutput{
			"kafkaBridgeTopicProduceUrl": kafkaBridgeTopicProduceUrl,
			"kafkaBridgeTopicAdminUrl": kafkaBridgeTopicAdminUrl,
		}, nil
	})
	deployment.AddStep("createInitTopics", "Create initial topics", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		spec := currentTopology.Spec
		kafkaBridgeTopicProduceUrl := c.GetStepOutput("deployStrimziKafkaBridge")["kafkaBridgeTopicProduceUrl"].(string)
		kafkaBridgeTopicAdminUrl := c.GetStepOutput("deployStrimziKafkaBridge")["kafkaBridgeTopicAdminUrl"].(string)
		var existingTopics []string
		err := common.GetHttpAsJsonWithResponse(kafkaBridgeTopicProduceUrl, true, &existingTopics)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		for _, topicToCreate := range spec.InitTopics {
			if stringsContains(existingTopics, topicToCreate.Name) {
				log.Printf("Topic %s already exists, do not create it again", topicToCreate.Name)
				continue
			}
			url := fmt.Sprintf("%s/%s", kafkaBridgeTopicAdminUrl, topicToCreate.Name)
			request := createTopicRequest{
				NumPartitions: topicToCreate.NumPartitions,
				ReplicationFactor: topicToCreate.ReplicationFactor,
			}
			var response createTopicResponse
			err := common.PostHttpAsJsonWithResponse(url, true, request, &response)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
		}
		return framework.NewDeploymentStepOutput(), nil
	})
	err = deployment.Run()
	if err != nil {
		return nil, err
	}
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*KafkaWithBridgeTopology)
	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)
	deployment, err := eks.CreateUninstallDeployment(currentTopology.Spec.EksSpec, commandEnvironment)
	if err != nil {
		return nil, err
	}
	deployment2 := kafkaonmsk.CreateUninstallDeployment(currentTopology.Spec.KafkaOnMskSpec)
	for _, step := range deployment2.GetSteps() {
		deployment.AddStep(step.GetName(), step.GetDescription(), step.GetDeployable())
	}
	err = deployment.Run()
	return deployment.GetOutput(), err
}

func stringsContains(slice []string, value string) bool {
	for _, entry := range slice {
		if entry == value {
			return true
		}
	}
	return false
}
