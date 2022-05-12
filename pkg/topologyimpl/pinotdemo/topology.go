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
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"gopkg.in/yaml.v3"
)

const (
	KindPinotDemoTopology = "PinotDemo"
)

type PinotDemoTopology struct {
	framework.TopologyBase `json:",inline" yaml:",inline"`
	Spec                   PinotDemoTopologySpec `json:"spec" yaml:"spec"`
}

type PinotDemoTopologySpec struct {
	Eks   eks.EksTopologySpec `json:"eks" yaml:"eks"`
	Pinot PinotComponentSpec `json:"pinot" yaml:"pinot"`
	Kafka KafkaComponentSpec `json:"kafka" yaml:"kafka"`
}

type PinotComponentSpec struct {
	HelmInstallName           string `json:"helmInstallName" yaml:"helmInstallName"`
	Namespace                 string `json:"namespace" yaml:"namespace"`
}

type KafkaComponentSpec struct {
	HelmInstallName           string `json:"helmInstallName" yaml:"helmInstallName"`
	Namespace                 string `json:"namespace" yaml:"namespace"`
}

func GeneratePinotDemoTopology() PinotDemoTopology {
	namePrefix := framework.DefaultNamePrefixTemplate
	s3BucketName := framework.DefaultS3BucketNameTemplate
	return CreateDefaultPinotDemoTopology(namePrefix, s3BucketName)
}

func CreateDefaultPinotDemoTopology(namePrefix string, s3BucketName string) PinotDemoTopology {
	topologyName := fmt.Sprintf("%s-kyuubi-on-eks-01", namePrefix)
	eksTopology := eks.CreateDefaultEksTopology(namePrefix, s3BucketName)
	topology := PinotDemoTopology{
		TopologyBase: framework.TopologyBase{
			ApiVersion: framework.DefaultVersion,
			Kind:       KindPinotDemoTopology,
			Metadata: framework.TopologyMetadata{
				Name: topologyName,
				CommandEnvironment: map[string]string{},
				Notes: map[string]string{},
			},
		},
		Spec: PinotDemoTopologySpec{
			Eks: eksTopology.Spec,
			Pinot: PinotComponentSpec{
				HelmInstallName: "pinot",
				Namespace: "pinot-demo-01",
			},
			Kafka: KafkaComponentSpec{
				HelmInstallName: "kafka",
				Namespace: "pinot-demo-01",
			},
		},
	}

	framework.CopyMissingKeyValuesFromStringMap(topology.Metadata.CommandEnvironment, eksTopology.Metadata.CommandEnvironment)

	return topology
}

func (t *PinotDemoTopology) GetKind() string {
	return t.Kind
}

func (t *PinotDemoTopology) GetMetadata() *framework.TopologyMetadata {
	return &t.Metadata
}

func (t *PinotDemoTopology) String() string {
	topologyBytes, err := yaml.Marshal(*t)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology as YAML: %s)", err.Error())
	}
	return string(topologyBytes)
}
