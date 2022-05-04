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
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"github.com/datapunchorg/punch/pkg/topologyimpl/kafkaonmsk"
)

const (

	KindKafkaTopology = "KafkaWithBridge"

	DefaultVersion    = "datapunch.org/v1alpha1"
	DefaultRegion     = "us-west-1"
	DefaultNamePrefix = "my"
	DefaultInstanceType                = "kafka.m5.large"

	CmdEnvKafkaBridgeHelmChart = "kafkaBridgeHelmChart"
)

type KafkaWithBridgeTopology struct {
	framework.TopologyBase `json:",inline" yaml:",inline"`
	Spec                   KafkaWithBridgeTopologySpec `json:"spec" yaml:"spec"`
}

type KafkaWithBridgeTopologySpec struct {
	KafkaOnMskSpec    kafkaonmsk.KafkaTopologySpec   `json:"kafkaOnMskSpec" yaml:"kafkaOnMskSpec"`
	EksSpec           eks.EksTopologySpec `json:"eksSpec" yaml:"eksSpec"`
	KafkaBridge   KafkaBridgeSpec  `json:"kafkaBridge" yaml:"kafkaBridge"`
}

type KafkaBridgeSpec struct {
	HelmInstallName string `json:"helmInstallName" yaml:"helmInstallName"`
	Namespace  string `json:"namespace" yaml:"namespace"`
	Image string `json:"image" yaml:"image"`
	KafkaBootstrapServers string `json:"kafkaBootstrapServers" yaml:"kafkaBootstrapServers"`
}

func GenerateDefaultTopology() KafkaWithBridgeTopology {
	namePrefix := "{{ or .Values.namePrefix `my` }}"
	s3BucketName := "{{ or .Values.s3BucketName .DefaultS3BucketName }}"
	return CreateDefaultTopology(namePrefix, s3BucketName)
}

func CreateDefaultTopology(namePrefix string, s3BucketName string) KafkaWithBridgeTopology {
	topologyName := fmt.Sprintf("%s-kafka-01", namePrefix)

	kafkaOnMskTopology := kafkaonmsk.CreateDefaultKafkaOnMskTopology(namePrefix)
	eksTopology := eks.CreateDefaultEksTopology(namePrefix, s3BucketName)

	topology := KafkaWithBridgeTopology{
		TopologyBase: framework.TopologyBase{
			ApiVersion: DefaultVersion,
			Kind:       KindKafkaTopology,
			Metadata: framework.TopologyMetadata{
				Name:               topologyName,
				CommandEnvironment: map[string]string{
					CmdEnvKafkaBridgeHelmChart: "third-party/helm-charts/strimzi/charts/strimzi-kafka-bridge-chart",
				},
				Notes:              map[string]string{},
			},
		},
		Spec: KafkaWithBridgeTopologySpec{
			KafkaOnMskSpec: kafkaOnMskTopology.Spec,
			EksSpec: eksTopology.Spec,
			KafkaBridge: KafkaBridgeSpec{
				HelmInstallName: "strimzi-kafka-bridge",
				Namespace: "kafka-01",
				Image: "ghcr.io/datapunchorg/strimzi-kafka-bridge:0.22.0-SNAPSHOT-1651610846",
				KafkaBootstrapServers: "localhost:9098",
			},
		},
	}

	framework.CopyMissingKeyValuesFromStringMap(topology.Metadata.CommandEnvironment, kafkaOnMskTopology.Metadata.CommandEnvironment)
	framework.CopyMissingKeyValuesFromStringMap(topology.Metadata.CommandEnvironment, eksTopology.Metadata.CommandEnvironment)

	return topology
}

func (t *KafkaWithBridgeTopology) GetKind() string {
	return t.Kind
}

func (t *KafkaWithBridgeTopology) GetMetadata() *framework.TopologyMetadata {
	return &t.Metadata
}
