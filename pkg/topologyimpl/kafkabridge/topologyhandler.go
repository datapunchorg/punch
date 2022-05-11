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

package kafkabridge

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/common"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/resource"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"github.com/datapunchorg/punch/pkg/topologyimpl/kafkaonmsk"
	"gopkg.in/yaml.v3"
	"log"
	"time"
)

func init() {
	framework.DefaultTopologyHandlerManager.AddHandler(KindKafkaTopology, &TopologyHandler{})
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
	currentTopology := topology.(*KafkaBridgeTopology)
	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)
	deployment := kafkaonmsk.CreateInstallDeployment(currentTopology.Spec.KafkaOnMskSpec)
	deployment.AddStep("checkEksCluster", "Check EKS cluster", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		result, err := resource.DescribeEksCluster(currentTopology.Spec.Region, currentTopology.Spec.EksClusterName)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		} else {
			log.Printf("Found EKS cluster %s in region %s", result.ClusterName, currentTopology.Spec.Region)
			return framework.NewDeploymentStepOutput(), nil
		}
	})
	deployment.AddStep("deployStrimziKafkaBridge", "Deploy Strimzi Kafka Bridge", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		spec := currentTopology.Spec
		if spec.KafkaBridge.KafkaBootstrapServers == "" {
			bootstrapServerString := c.GetStepOutput("createKafkaCluster")["bootstrapServerString"].(string)
			spec.KafkaBridge.KafkaBootstrapServers = bootstrapServerString
			log.Printf("Set spec.KafkaBridge.KafkaBootstrapServers: %s", bootstrapServerString)
		}
		urls, err := resource.GetEksNginxLoadBalancerUrls(commandEnvironment, spec.Region, spec.EksClusterName, spec.NginxNamespace, spec.NginxServiceName, eks.NodePortLocalHttps)
		if err != nil {
			log.Fatalf("Failed to get NGINX load balancer urls: %s", err.Error())
		}
		loadBalancerUrl := resource.GetLoadBalancerPreferredUrl(urls)
		if loadBalancerUrl == "" {
			return framework.NewDeploymentStepOutput(), fmt.Errorf("did not find load balancer url")
		}
		err = DeployKafkaBridge(commandEnvironment, spec)
		if err != nil {
			return framework.NewDeploymentStepOutput(), fmt.Errorf("failed to install kafka bridge service: %s", err.Error())
		}
		kafkaBridgeUrl := fmt.Sprintf("%s/topics", loadBalancerUrl)
		maxWaitMinutes := 10
		err = common.WaitHttpUrlReady(kafkaBridgeUrl, time.Duration(maxWaitMinutes) * time.Minute, 10 * time.Second)
		if err != nil {
			return framework.NewDeploymentStepOutput(), fmt.Errorf("kafka bridge url %s is not ready after waiting %d minutes: %s", kafkaBridgeUrl, maxWaitMinutes, err.Error())
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
	err := deployment.Run()
	if err != nil {
		return nil, err
	}
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*KafkaBridgeTopology)
	deployment := kafkaonmsk.CreateUninstallDeployment(currentTopology.Spec.KafkaOnMskSpec)
	// TODO delete kafka bridge service
	err := deployment.Run()
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
