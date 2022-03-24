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

package sparkonk8s

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"gopkg.in/yaml.v3"
)

const (
	DefaultApiUserName                  = "user1"
	DefaultOperatorImageRepository      = "ghcr.io/datapunchorg/spark-on-k8s-operator"
	DefaultSparkOperatorImageTag        = "master-datapunch"
	DefaultSparkOperatorNamespace       = "spark-operator-01"
	DefaultSparkOperatorHelmInstallName = "spark-operator-01"
	DefaultSparkApplicationNamespace    = "spark-01"

	KindSparkTopology = "SparkOnK8s"

	FieldMaskValue = "***"

	CmdEnvSparkOperatorHelmChart = "sparkOperatorHelmChart"

	DefaultVersion    = "datapunch.org/v1alpha1"
	DefaultNamePrefix = "my"
)

type SparkTopology struct {
	ApiVersion string                     `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                     `json:"kind" yaml:"kind"`
	Metadata   framework.TopologyMetadata `json:"metadata"`
	Spec       SparkTopologySpec          `json:"spec"`
}

type SparkTopologySpec struct {
	EksSpec       eks.EksTopologySpec `json:"eksSpec" yaml:"eksSpec"`
	SparkOperator SparkOperator       `json:"sparkOperator" yaml:"sparkOperator"`
	ApiGateway    SparkApiGateway     `json:"apiGateway" yaml:"apiGateway"`
}

type SparkApiGateway struct {
	UserName     string `json:"userName" yaml:"userName"`
	UserPassword string `json:"userPassword" yaml:"userPassword"`
}

type SparkOperator struct {
	HelmInstallName           string `json:"helmInstallName" yaml:"helmInstallName"`
	Namespace                 string `json:"namespace" yaml:"namespace"`
	ImageRepository           string `json:"imageRepository" yaml:"imageRepository"`
	ImageTag                  string `json:"imageTag" yaml:"imageTag"`
	SparkApplicationNamespace string `json:"sparkApplicationNamespace" yaml:"sparkApplicationNamespace"`
}

func CreateDefaultSparkTopology(namePrefix string, s3BucketName string) SparkTopology {
	topologyName := fmt.Sprintf("%s-spark-k8s", namePrefix)
	topology := SparkTopology{
		ApiVersion: DefaultVersion,
		Kind:       KindSparkTopology,
		Metadata: framework.TopologyMetadata{
			Name: topologyName,
			CommandEnvironment: map[string]string{
				eks.CmdEnvHelmExecutable: eks.DefaultHelmExecutable,
			},
			Notes: map[string]string{},
		},
		Spec: SparkTopologySpec{
			EksSpec: eks.CreateDefaultEksTopology(namePrefix, s3BucketName).Spec,
			ApiGateway: SparkApiGateway{
				UserName: DefaultApiUserName,
			},
			SparkOperator: SparkOperator{
				HelmInstallName:           DefaultSparkOperatorHelmInstallName,
				ImageRepository:           DefaultOperatorImageRepository,
				ImageTag:                  DefaultSparkOperatorImageTag,
				Namespace:                 DefaultSparkOperatorNamespace,
				SparkApplicationNamespace: DefaultSparkApplicationNamespace,
			},
		},
	}
	eks.UpdateEksTopologyByS3BucketName(&topology.Spec.EksSpec, s3BucketName)
	return topology
}

func (t *SparkTopology) GetKind() string {
	return t.Kind
}

func (t *SparkTopology) GetSpec() framework.TopologySpec {
	return t.Spec
}

func (t *SparkTopology) ToString() string {
	topologyBytes, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology: %s)", err.Error())
	}
	var copy SparkTopology
	err = yaml.Unmarshal(topologyBytes, &copy)
	if err != nil {
		return fmt.Sprintf("(Failed to deserialize topology in ToYamlString(): %s)", err.Error())
	}
	if copy.Spec.ApiGateway.UserPassword != "" {
		copy.Spec.ApiGateway.UserPassword = FieldMaskValue
	}
	topologyBytes, err = yaml.Marshal(copy)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology in ToYamlString(): %s)", err.Error())
	}
	return string(topologyBytes)
}
