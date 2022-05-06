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
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"github.com/datapunchorg/punch/pkg/topologyimpl/kafkaonmsk"
)

const (
	KindKafkaTopology = "KafkaBridge"

	CmdEnvKafkaBridgeHelmChart = "kafkaBridgeHelmChart"
)

type KafkaBridgeTopology struct {
	framework.TopologyBase `json:",inline" yaml:",inline"`
	Spec                   KafkaBridgeTopologySpec `json:"spec" yaml:"spec"`
}

type KafkaBridgeTopologySpec struct {
	NamePrefix        string                   `json:"namePrefix" yaml:"namePrefix"`
	Region            string                   `json:"region" yaml:"region"`
	KafkaOnMskSpec    kafkaonmsk.KafkaTopologySpec   `json:"kafkaOnMskSpec" yaml:"kafkaOnMskSpec"`
	EksClusterName    string                   `json:"eksClusterName" yaml:"eksClusterName"`
	EksVpcId    string                          `json:"eksVpcId" yaml:"eksVpcId"`
	NginxNamespace    string                    `json:"nginxNamespace" yaml:"nginxNamespace"`
	NginxServiceName    string                    `json:"nginxServiceName" yaml:"nginxServiceName"`
	KafkaBridge   KafkaBridgeSpec  `json:"kafkaBridge" yaml:"kafkaBridge"`
	InitTopics    []KafkaTopic  `json:"initTopics" yaml:"initTopics"`
}

type KafkaBridgeSpec struct {
	HelmInstallName string `json:"helmInstallName" yaml:"helmInstallName"`
	Namespace  string `json:"namespace" yaml:"namespace"`
	Image string `json:"image" yaml:"image"`
	KafkaBootstrapServers string `json:"kafkaBootstrapServers" yaml:"kafkaBootstrapServers"`
}

type KafkaTopic struct {
	Name string `json:"name" yaml:"name"`
	NumPartitions int64  `json:"numPartitions" yaml:"numPartitions"`
	ReplicationFactor int64  `json:"replicationFactor" yaml:"replicationFactor"`
}

func GenerateDefaultTopology() KafkaBridgeTopology {
	namePrefix := framework.DefaultNamePrefixTemplate
	s3BucketName := framework.DefaultS3BucketNameTemplate
	return CreateDefaultTopology(namePrefix, s3BucketName)
}

func CreateDefaultTopology(namePrefix string, s3BucketName string) KafkaBridgeTopology {
	topologyName := fmt.Sprintf("%s-kafka-01", namePrefix)

	kafkaOnMskTopology := kafkaonmsk.CreateDefaultKafkaOnMskTopology(namePrefix)
	eksTopology := eks.CreateDefaultEksTopology(namePrefix, s3BucketName)

	topology := KafkaBridgeTopology{
		TopologyBase: framework.TopologyBase{
			ApiVersion: framework.DefaultVersion,
			Kind:       KindKafkaTopology,
			Metadata: framework.TopologyMetadata{
				Name:               topologyName,
				CommandEnvironment: map[string]string{
					framework.CmdEnvHelmExecutable: framework.DefaultHelmExecutable,
					CmdEnvKafkaBridgeHelmChart: "third-party/helm-charts/strimzi/charts/strimzi-kafka-bridge-chart",
				},
				Notes:              map[string]string{},
			},
		},
		Spec: KafkaBridgeTopologySpec{
			NamePrefix:   namePrefix,
			Region:       fmt.Sprintf("{{ or .Values.region `%s` }}", framework.DefaultRegion),
			KafkaOnMskSpec: kafkaOnMskTopology.Spec,
			EksClusterName: eksTopology.Spec.Eks.ClusterName,
			EksVpcId: eksTopology.Spec.VpcId,
			NginxNamespace: eks.DefaultNginxIngressNamespace,
			NginxServiceName: eks.DefaultNginxServiceName,
			KafkaBridge: KafkaBridgeSpec{
				HelmInstallName: "strimzi-kafka-bridge",
				Namespace: "kafka-01",
				Image: "ghcr.io/datapunchorg/strimzi-kafka-bridge:0.22.0-snapshot-1651702291",
				KafkaBootstrapServers: "",
			},
			InitTopics: []KafkaTopic {
				{
					Name: "topic_01",
					NumPartitions: 1,
					ReplicationFactor: 1,
				},
				{
					Name: "topic_02",
					NumPartitions: 2,
					ReplicationFactor: 2,
				},
			},
		},
	}

	framework.CopyMissingKeyValuesFromStringMap(topology.Metadata.CommandEnvironment, kafkaOnMskTopology.Metadata.CommandEnvironment)
	framework.CopyMissingKeyValuesFromStringMap(topology.Metadata.CommandEnvironment, eksTopology.Metadata.CommandEnvironment)

	return topology
}

func (t *KafkaBridgeTopology) GetKind() string {
	return t.Kind
}

func (t *KafkaBridgeTopology) GetMetadata() *framework.TopologyMetadata {
	return &t.Metadata
}
