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

package superset

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"gopkg.in/yaml.v3"
)

const (
	KindSupersetTopology = "Superset"

	CmdEnvSupersetHelmChart = "supersetHelmChart"
)

type SupersetTopology struct {
	framework.TopologyBase `json:",inline" yaml:",inline"`
	Spec                   SupersetTopologySpec `json:"spec" yaml:"spec"`
}

type SupersetTopologySpec struct {
	NamePrefix      string                    `json:"namePrefix" yaml:"namePrefix"`
	Region          string                    `json:"region" yaml:"region"`
	EksClusterName  string `json:"eksClusterName" yaml:"eksClusterName"`
	HelmInstallName           string `json:"helmInstallName" yaml:"helmInstallName"`
	Namespace                 string `json:"namespace" yaml:"namespace"`
	InitDatabases   []DatabaseInfo `json:"initDatabases" yaml:"initDatabases"`
}

type DatabaseInfo struct {
	SqlalchemyUri           string `json:"sqlalchemyUri" yaml:"sqlalchemyUri"`
}

func GenerateSupersetTopology() SupersetTopology {
	namePrefix := framework.DefaultNamePrefixTemplate
	s3BucketName := framework.DefaultS3BucketNameTemplate
	return CreateDefaultSupersetTopology(namePrefix, s3BucketName)
}

func CreateDefaultSupersetTopology(namePrefix string, s3BucketName string) SupersetTopology {
	topologyName := fmt.Sprintf("%s-superset-01", namePrefix)
	eksTopology := eks.CreateDefaultEksTopology(namePrefix, s3BucketName)
	topology := SupersetTopology{
		TopologyBase: framework.TopologyBase{
			ApiVersion: framework.DefaultVersion,
			Kind:       KindSupersetTopology,
			Metadata: framework.TopologyMetadata{
				Name: topologyName,
				CommandEnvironment: map[string]string{
					CmdEnvSupersetHelmChart: "third-party/helm-charts/superset/charts/superset",
				},
				Notes: map[string]string{},
			},
		},
		Spec: SupersetTopologySpec{
			NamePrefix: namePrefix,
			Region: eksTopology.Spec.Region,
			EksClusterName: eksTopology.Spec.EksCluster.ClusterName,
			HelmInstallName: "superset",
			Namespace: "superset-01",
			InitDatabases: []DatabaseInfo{
				{
					SqlalchemyUri: "",
				},
			},
		},
	}
	return topology
}

func (t *SupersetTopology) GetKind() string {
	return t.Kind
}

func (t *SupersetTopology) GetMetadata() *framework.TopologyMetadata {
	return &t.Metadata
}

func (t *SupersetTopology) String() string {
	topologyBytes, err := yaml.Marshal(*t)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology as YAML: %s)", err.Error())
	}
	return string(topologyBytes)
}
