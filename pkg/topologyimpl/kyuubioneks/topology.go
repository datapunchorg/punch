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

package kyuubioneks

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/topologyimpl/sparkoneks"

	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"gopkg.in/yaml.v3"
)

const (
	KindKyuubiOnEksTopology = "KyuubiOnEks"

	CmdEnvKyuubiHelmChart = "kyuubiHelmChart"
)

type KyuubiOnEksTopology struct {
	framework.TopologyBase `json:",inline" yaml:",inline"`
	Spec                   KyuubiOnEksTopologySpec `json:"spec" yaml:"spec"`
}

type KyuubiOnEksTopologySpec struct {
	Eks   eks.EksTopologySpec `json:"eks" yaml:"eks"`
	Spark sparkoneks.SparkComponentSpec  `json:"spark" yaml:"spark"`
	Kyuubi KyuubiComponentSpec `json:"kyuubi" yaml:"kyuubi"`
}

type KyuubiComponentSpec struct {
	HelmInstallName           string `json:"helmInstallName" yaml:"helmInstallName"`
	Namespace                 string `json:"namespace" yaml:"namespace"`
	ImageRepository           string `json:"imageRepository" yaml:"imageRepository"`
	ImageTag                  string `json:"imageTag" yaml:"imageTag"`
	SparkSqlEngine            SparkSqlEngineSpec `json:"sparkSqlEngine" yaml:"sparkSqlEngine"`
}

type SparkSqlEngineSpec struct {
	JarFile string `json:"jarFile" yaml:"jarFile"`
}

func GenerateKyuubiOnEksTopology() KyuubiOnEksTopology {
	namePrefix := framework.DefaultNamePrefixTemplate
	s3BucketName := framework.DefaultS3BucketNameTemplate
	return CreateDefaultKyuubiOnEksTopology(namePrefix, s3BucketName)
}

func CreateDefaultKyuubiOnEksTopology(namePrefix string, s3BucketName string) KyuubiOnEksTopology {
	topologyName := fmt.Sprintf("%s-kyuubi-on-eks-01", namePrefix)
	sparkOnEksTopology := sparkoneks.CreateDefaultSparkOnEksTopology(namePrefix, s3BucketName)
	topology := KyuubiOnEksTopology{
		TopologyBase: framework.TopologyBase{
			ApiVersion: framework.DefaultVersion,
			Kind:       KindKyuubiOnEksTopology,
			Metadata: framework.TopologyMetadata{
				Name: topologyName,
				CommandEnvironment: map[string]string{
					CmdEnvKyuubiHelmChart: "third-party/helm-charts/kyuubi/charts/kyuubi",
				},
				Notes: map[string]string{},
			},
		},
		Spec: KyuubiOnEksTopologySpec{
			Eks: sparkOnEksTopology.Spec.Eks,
			Spark: sparkOnEksTopology.Spec.Spark,
			Kyuubi: KyuubiComponentSpec{
				HelmInstallName: "kyuubi",
				Namespace: "kyuubi-01",
				ImageRepository: "ghcr.io/datapunchorg/incubator-kyuubi",
				ImageTag: "kyuubi-1652420363",
				SparkSqlEngine: SparkSqlEngineSpec{
					JarFile: "s3a://datapunch-public-01/jars/kyuubi-spark-sql-engine_2.12-1.5.0-incubating.jar",
				},
			},
		},
	}

	framework.CopyMissingKeyValuesFromStringMap(topology.Metadata.CommandEnvironment, sparkOnEksTopology.Metadata.CommandEnvironment)

	return topology
}

func (t *KyuubiOnEksTopology) GetKind() string {
	return t.Kind
}

func (t *KyuubiOnEksTopology) GetMetadata() *framework.TopologyMetadata {
	return &t.Metadata
}

func (t *KyuubiOnEksTopology) String() string {
	copy := *t
	if copy.Spec.Spark.Gateway.Password != "" {
		copy.Spec.Spark.Gateway.Password = framework.FieldMaskValue
	}
	topologyBytes, err := yaml.Marshal(copy)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology as YAML: %s)", err.Error())
	}
	return string(topologyBytes)
}
