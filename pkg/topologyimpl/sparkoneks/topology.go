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

package sparkoneks

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"gopkg.in/yaml.v3"
)

const (
	DefaultApiUserName                  = "user1"
	DefaultOperatorImageRepository      = "ghcr.io/datapunchorg/spark-on-k8s-operator"
	DefaultSparkOperatorImageTag        = "sha-9f1794f"
	DefaultSparkOperatorNamespace       = "spark-operator-01"
	DefaultSparkOperatorHelmInstallName = "spark-operator-01"
	DefaultSparkApplicationNamespace    = "spark-01"
	DefaultSparkHistoryServerHelmInstallName = "spark-history-server"
	DefaultSparkHistoryServerImageRepository = "ghcr.io/datapunchorg/spark-on-k8s-operator"
	DefaultSparkHistoryServerImageTag        = "spark-history-server-3.2-1650337377"
	DefaultSparkHistoryServerNamespace       = "spark-history-server"

	KindSparkTopology = "SparkOnEks"

	FieldMaskValue = "***"

	CmdEnvSparkOperatorHelmChart = "sparkOperatorHelmChart"
	CmdEnvHistoryServerHelmChart = "historyServerHelmChart"

	DefaultVersion    = "datapunch.org/v1alpha1"
	DefaultNamePrefix = "my"
)

type SparkTopology struct {
	framework.TopologyBase               `json:",inline" yaml:",inline"`
	Spec       SparkTopologySpec          `json:"spec" yaml:"spec"`
}

type SparkTopologySpec struct {
	EksSpec           eks.EksTopologySpec `json:"eksSpec" yaml:"eksSpec"`
	SparkOperator SparkOperator       `json:"sparkOperator" yaml:"sparkOperator"`
	ApiGateway    SparkApiGateway     `json:"apiGateway" yaml:"apiGateway"`
	HistoryServer SparkHistoryServer       `json:"historyServer" yaml:"historyServer"`
}

type SparkApiGateway struct {
	UserName     string `json:"userName" yaml:"userName"`
	UserPassword string `json:"userPassword" yaml:"userPassword"`
	SparkEventLogDir string `json:"sparkEventLogDir" yaml:"sparkEventLogDir"`
}

type SparkOperator struct {
	HelmInstallName           string `json:"helmInstallName" yaml:"helmInstallName"`
	Namespace                 string `json:"namespace" yaml:"namespace"`
	ImageRepository           string `json:"imageRepository" yaml:"imageRepository"`
	ImageTag                  string `json:"imageTag" yaml:"imageTag"`
	SparkApplicationNamespace string `json:"sparkApplicationNamespace" yaml:"sparkApplicationNamespace"`
}

type SparkHistoryServer struct {
	HelmInstallName           string `json:"helmInstallName" yaml:"helmInstallName"`
	Namespace                 string `json:"namespace" yaml:"namespace"`
	ImageRepository           string `json:"imageRepository" yaml:"imageRepository"`
	ImageTag                  string `json:"imageTag" yaml:"imageTag"`
}

func GenerateSparkTopology() SparkTopology {
	namePrefix := "{{ or .Values.namePrefix `my` }}"
	s3BucketName := "{{ or .Values.s3BucketName .DefaultS3BucketName }}"
	return CreateDefaultSparkTopology(namePrefix, s3BucketName)
}

func CreateDefaultSparkTopology(namePrefix string, s3BucketName string) SparkTopology {
	topologyName := fmt.Sprintf("%s-spark-k8s", namePrefix)
	eksTopology := eks.CreateDefaultEksTopology(namePrefix, s3BucketName)
	topology := SparkTopology{
		TopologyBase: framework.TopologyBase{
			ApiVersion: DefaultVersion,
			Kind:       KindSparkTopology,
			Metadata: framework.TopologyMetadata{
				Name: topologyName,
				CommandEnvironment: map[string]string{
					CmdEnvSparkOperatorHelmChart: "third-party/helm-charts/spark-operator-service/charts/spark-operator-chart",
					CmdEnvHistoryServerHelmChart: "third-party/helm-charts/spark-history-server/charts/spark-history-server-chart",
				},
				Notes: map[string]string{},
			},
		},
		Spec: SparkTopologySpec{
			EksSpec: eksTopology.Spec,
			ApiGateway: SparkApiGateway{
				UserName: DefaultApiUserName,
				SparkEventLogDir: fmt.Sprintf("s3a://%s/punch/%s/sparkEventLog", s3BucketName, namePrefix),
			},
			SparkOperator: SparkOperator{
				HelmInstallName:           DefaultSparkOperatorHelmInstallName,
				ImageRepository:           DefaultOperatorImageRepository,
				ImageTag:                  DefaultSparkOperatorImageTag,
				Namespace:                 DefaultSparkOperatorNamespace,
				SparkApplicationNamespace: DefaultSparkApplicationNamespace,
			},
			HistoryServer: SparkHistoryServer{
				HelmInstallName:           DefaultSparkHistoryServerHelmInstallName,
				ImageRepository:           DefaultSparkHistoryServerImageRepository,
				ImageTag:                  DefaultSparkHistoryServerImageTag,
				Namespace:                 DefaultSparkHistoryServerNamespace,
			},
		},
	}

	framework.CopyMissingKeyValuesFromStringMap(topology.Metadata.CommandEnvironment, eksTopology.Metadata.CommandEnvironment)

	eks.UpdateEksTopologyByS3BucketName(&topology.Spec.EksSpec, s3BucketName)
	return topology
}

func (t *SparkTopology) GetKind() string {
	return t.Kind
}

func (t *SparkTopology) GetMetadata() *framework.TopologyMetadata {
	return &t.Metadata
}

func (t *SparkTopology) String() string {
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