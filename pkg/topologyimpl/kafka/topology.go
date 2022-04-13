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
	"github.com/datapunchorg/punch/pkg/resource"
	"gopkg.in/yaml.v3"
)

const (
	DefaultUserName = "user1"

	KindKafkaTopology = "Kafka"

	FieldMaskValue = "***"

	DefaultVersion    = "datapunch.org/v1alpha1"
	DefaultRegion     = "us-west-1"
	DefaultNamePrefix = "my"
	DefaultInstanceType                = "kafka.m5.large"
)

type KafkaTopology struct {
	ApiVersion string                     `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                     `json:"kind" yaml:"kind"`
	Metadata framework.TopologyMetadata `json:"metadata"`
	Spec     KafkaTopologySpec          `json:"spec"`
}

type KafkaTopologySpec struct {
	NamePrefix        string   `json:"namePrefix" yaml:"namePrefix"`
	Region            string   `json:"region" yaml:"region"`
	VpcId             string   `json:"vpcId" yaml:"vpcId"`
	ClusterName       string   `json:"clusterName" yaml:"clusterName"`
	SubnetIds         []string                  `json:"subnetIds" yaml:"subnetIds"`
	KafkaVersion      string   `json:"kafkaVersion" yaml:"kafkaVersion"`
	SecurityGroups    []resource.SecurityGroup `json:"securityGroups" yaml:"securityGroups"`
	BrokerStorageGB   int64    `json:"brokerStorageGB" yaml:"brokerStorageGB"`
}

func CreateDefaultKafkaTopology(namePrefix string) KafkaTopology {
	topologyName := fmt.Sprintf("%s-kafka-01", namePrefix)
	securityGroupName := fmt.Sprintf("%s-kafka-sg-01", namePrefix)
	topology := KafkaTopology{
		ApiVersion: DefaultVersion,
		Kind:       KindKafkaTopology,
		Metadata: framework.TopologyMetadata{
			Name:               topologyName,
			CommandEnvironment: map[string]string{},
			Notes:              map[string]string{},
		},
		Spec: KafkaTopologySpec{
			NamePrefix:         namePrefix,
			Region:             DefaultRegion,
			VpcId:              "{{ or .Values.vpcId .DefaultVpcId }}",
			ClusterName:         topologyName,
			KafkaVersion:       "2.8.1",
			SecurityGroups: []resource.SecurityGroup{
				{
					Name: securityGroupName,
					InboundRules: []resource.SecurityGroupInboundRule{
						{
							IPProtocol: "-1",
							FromPort:   -1,
							ToPort:     -1,
							IPRanges:   []string{"0.0.0.0/0"},
						},
					},
				},
			},
			BrokerStorageGB: 20,
		},
	}
	return topology
}

func (t *KafkaTopology) GetKind() string {
	return t.Kind
}

func (t *KafkaTopology) GetSpec() framework.TopologySpec {
	return t.Spec
}

func (t *KafkaTopology) ToString() string {
	topologyBytes, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology: %s)", err.Error())
	}
	return string(topologyBytes)
}
