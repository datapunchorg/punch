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

package kafkaonmsk

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/resource"
)

const (

	KindKafkaOnMskTopology = "KafkaOnMsk"

	DefaultVersion    = "datapunch.org/v1alpha1"
	DefaultRegion     = "us-west-1"
	DefaultNamePrefix = "my"
	DefaultInstanceType                = "kafka.m5.large"
)

type KafkaTopology struct {
	framework.TopologyBase               `json:",inline" yaml:",inline"`
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
	NumBrokers        int64   `json:"numBrokers" yaml:"numBrokers"`
	BrokerStorageGB   int64    `json:"brokerStorageGB" yaml:"brokerStorageGB"`
}

func CreateDefaultKafkaOnMskTopology(namePrefix string) KafkaTopology {
	topologyName := fmt.Sprintf("%s-msk-01", namePrefix)
	securityGroupName := fmt.Sprintf("%s-msk-sg-01", namePrefix)
	region := DefaultRegion
	vpcId, err := awslib.GetDefaultVpcId(region)
	if err != nil {
		vpcId = fmt.Sprintf("INVALID: %s", err.Error())
	}
	subnetIds, err := resource.GetSubnetIds(region, vpcId)
	if err != nil {
		subnetIds = []string{fmt.Sprintf("INVALID: %s", err.Error())}
	}
	topology := KafkaTopology{
		TopologyBase: framework.TopologyBase{
			ApiVersion: DefaultVersion,
			Kind:       KindKafkaOnMskTopology,
			Metadata: framework.TopologyMetadata{
				Name:               topologyName,
				CommandEnvironment: map[string]string{},
				Notes:              map[string]string{},
			},
		},
		Spec: KafkaTopologySpec{
			NamePrefix:         namePrefix,
			Region:             region,
			VpcId:              vpcId,
			ClusterName:        topologyName,
			SubnetIds:          subnetIds,
			KafkaVersion:       "2.8.1",
			SecurityGroups: []resource.SecurityGroup{
				{
					Name: securityGroupName,
					InboundRules: []resource.SecurityGroupInboundRule{
						{
							IPProtocol: "-1",
							FromPort:   -1,
							ToPort:     -1,
							IPRanges:   []string{"{{ or .Values.vpcCidrBlock .DefaultVpcCidrBlock }}"},
						},
					},
				},
			},
			NumBrokers: int64(len(subnetIds)),
			BrokerStorageGB: 20,
		},
	}
	return topology
}

func (t *KafkaTopology) GetKind() string {
	return t.Kind
}

func (t *KafkaTopology) GetMetadata() *framework.TopologyMetadata {
	return &t.Metadata
}
